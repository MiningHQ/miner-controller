/*
  MiningHQ Miner Controller - manages cryptocurrency miners on a user's machine.
  https://mininghq.io

  Copyright (C) 2018  Donovan Solms     <https://github.com/donovansolms>

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package ctl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/donovansolms/mininghq-miner-controller/src/mhq"
	"github.com/donovansolms/mininghq-miner-controller/src/miner"
	"github.com/donovansolms/mininghq-rpcproto/rpcproto"
	"github.com/donovansolms/rich-go/client"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Ctl implements the core command and control functionality to communicate
// with MiningHQ and manage the miners on the local rig
type Ctl struct {
	// mutex for protecting the miner slice
	mutex sync.Mutex
	// rigID is this rig's identifier
	rigID string
	// websocketEndpoint is the command websocket API endpoint
	websocketEndpoint string
	// grpcEndpoint is the endpoint to bind for the
	// miner manager to use
	// Must be localhost
	grpcEndpoint string
	// grpcServer is the local manager API server
	grpcServer *grpc.Server
	// miningKey is the unique key for this user's account
	miningKey string
	// miners hold the current active miners
	miners []miner.Miner
	// currentState of this rig
	currentState rpcproto.MinerState
	// currentAssignment is the current mining assignment
	currentAssignment *rpcproto.RigAssignmentRequest
	// currentInfo holds the current rig information
	currentInfo *rpcproto.RigInfoResponse
	// client for communicating with MiningHQ
	client *mhq.WebSocketClient
	// log for logs :)
	log *logrus.Entry
}

// New creates a new instance of the core controller
func New(
	websocketEndpoint string,
	grpcEndpoint string,
	miningKey string,
	rigID string,
	log *logrus.Entry,
) (*Ctl, error) {

	ctl := Ctl{
		rigID:             rigID,
		websocketEndpoint: websocketEndpoint,
		grpcEndpoint:      grpcEndpoint,
		miningKey:         miningKey,
		log:               log,
	}

	// Create the gRPC manager API
	serverOptions := []grpc.ServerOption{}
	ctl.grpcServer = grpc.NewServer(serverOptions...)
	rpcproto.RegisterManagerServiceServer(ctl.grpcServer, &ctl)

	return &ctl, nil
}

// Run the core controller
func (ctl *Ctl) Run() error {
	ctl.log.Info("Started")

	var err error
	// This loop retries forever to connect. We'll only ever execute this
	// more than once if MiningHQ is down
	for {
		ctl.log.Info("Connecting to MiningHQ services")
		// NewWebSocketClient connects to the given endpoint and authenticates
		ctl.client, err = mhq.NewWebSocketClient(
			ctl.websocketEndpoint,
			ctl.miningKey,
			ctl.rigID,
			ctl.onMessage)
		if err == nil {
			ctl.log.Info("Connected to MiningHQ services")
			break
		}

		ctl.log.Warningf("Unable to connect to MiningHQ services: %s", err)
		ctl.log.Warning("Retrying in 10 seconds...")
		time.Sleep(time.Second * 10)
	}

	// Setup signal handlers
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	// Remember, on Linux, syscall.SIGKILL can't be caught
	go func() {
		sig := <-signalChannel
		switch sig {
		case syscall.SIGHUP:
			ctl.Stop()
		case syscall.SIGINT:
			ctl.Stop()
		case syscall.SIGTERM:
			ctl.Stop()
		}
	}()

	// TODO: Send current rig specs to MiningHQ
	// TODO: Should this even be done?
	// systemInfo, err := caps.GetSystemInfo()
	// if err != nil {
	// 	return err
	// }
	// packet := spec.WSPacket{
	// 	Message: &spec.WSPacket_SystemInfo{
	// 		SystemInfo: systemInfo,
	// 	},
	// }
	// ctl.sendMessage(&packet)

	// Send initial request for rig information
	packet := rpcproto.Packet{
		Method: rpcproto.Method_RigInfo,
		Params: &rpcproto.Packet_RigInfoRequest{
			RigInfoRequest: &rpcproto.RigInfoRequest{
				RigID: ctl.rigID,
			},
		},
	}
	err = ctl.sendMessage(&packet)
	if err != nil {
		ctl.log.WithFields(logrus.Fields{
			"rig_id": ctl.rigID,
			"method": packet.Method,
		}).Errorf("Unable to query rig info: %s", err)
	}

	// Start the gRPC manager API
	listener, err := net.Listen("tcp", ctl.grpcEndpoint)
	if err != nil {
		ctl.log.WithFields(logrus.Fields{
			"endpoint": ctl.grpcEndpoint,
		}).Errorf("Unable to start listener for Manager API server: %s", err)

		return err
	}

	ctl.log.WithFields(logrus.Fields{
		"endpoint": ctl.grpcEndpoint,
	}).Info("gRPC API server starting")

	go func() {
		err = ctl.grpcServer.Serve(listener)
		if err != nil {
			ctl.log.WithFields(logrus.Fields{
				"endpoint": ctl.grpcEndpoint,
			}).Errorf("Unable to start gRPC Manager API server: %s", err)
		}
	}()

	// Once our connection is processed by MiningHQ, we'll
	// receive the RigAssignment and start mining - if the user's account
	// is set up for that
	err = ctl.client.Start()
	if err != nil {
		switch typedErr := err.(type) {
		case *websocket.CloseError:
			if typedErr.Code != websocket.CloseNormalClosure {
				err = ctl.Stop()
				if err != nil {
					ctl.log.Debugf("Error during stopping: %s", err)
				}
				return ctl.Run()
			}
		default:
			return err
		}
	}
	return nil
}

// onMessage is called when a new message is received via the websocket
func (ctl *Ctl) onMessage(data []byte, err error) error {

	if err != nil {
		switch err.(type) {
		case *websocket.CloseError:
			ctl.log.Debugf("WebSocket closing: %s", err)
		default:
			ctl.log.Errorf("WebSocket read error: %s", err)
		}
		// TODO: ctl.Stop() ?
		return err
	}

	var packet rpcproto.Packet
	err = proto.Unmarshal(data, &packet)
	if err != nil {
		ctl.log.Warningf("Unable to process message: %s", err)
		return err
	}

	switch packet.Method {
	//
	// Handle incoming warnings
	//
	case rpcproto.Method_RigWarning:
		// TODO: Implement warnings from MiningHQ

	//
	// Handle incoming state update requests
	//
	case rpcproto.Method_State:
		request := packet.GetStateRequest()
		if request == nil {
			ctl.log.WithFields(logrus.Fields{
				"method": packet.Method.String(),
				"params": "StateRequest",
			}).Error("Params are nil")
			return errors.New("params are nil")
		}

		ctl.log.WithFields(logrus.Fields{
			"method": packet.Method.String(),
			"params": "StateRequest",
		}).Debug("New RPC message processing")

		err = ctl.handleControl(request)
		if err != nil {
			ctl.log.Errorf("Unable to update state: %s", err)
			// Send response message
			response := rpcproto.Packet{
				Method: rpcproto.Method_State,
				Params: &rpcproto.Packet_StateResponse{
					StateResponse: &rpcproto.StateResponse{
						Status:     "StateResponse error",
						StatusCode: http.StatusInternalServerError,
						Reason:     fmt.Sprintf("Unable to update rig state: %s", err),
					},
				},
			}
			err = ctl.sendMessage(&response)
			if err != nil {
				ctl.log.Errorf("Unable to send StateResponse to MiningHQ: %s", err)
			}
			return err
		}
		ctl.log.Info("Rig state has been updated")

		// Send response message
		response := rpcproto.Packet{
			Method: rpcproto.Method_State,
			Params: &rpcproto.Packet_StateResponse{
				StateResponse: &rpcproto.StateResponse{
					Status:     "Ok",
					StatusCode: http.StatusOK,
				},
			},
		}
		err = ctl.sendMessage(&response)
		if err != nil {
			ctl.log.Errorf("Unable to send StateResponse to MiningHQ: %s", err)
		}

	//
	// Handle incoming requests for logs
	//
	case rpcproto.Method_Logs:

		request := packet.GetLogsRequest()
		if request == nil {
			ctl.log.WithFields(logrus.Fields{
				"method": packet.Method.String(),
				"params": "LogRequest",
			}).Error("Params are nil")
			return errors.New("params are nil")
		}

		ctl.log.WithFields(logrus.Fields{
			"method": packet.Method.String(),
			"params": "LogRequest",
		}).Debug("New RPC message processing")

		logsResponse := rpcproto.LogsResponse{
			MinerLogs: ctl.getMinersLogs(),
		}

		// Send the logs back
		response := rpcproto.Packet{
			Method: rpcproto.Method_Logs,
			Params: &rpcproto.Packet_LogsResponse{
				LogsResponse: &logsResponse,
			},
		}
		err = ctl.sendMessage(&response)
		if err != nil {
			ctl.log.Errorf("Unable to send LogsResponse to MiningHQ: %s", err)
		}
		ctl.log.Info("Rig logs sent")

	//
	// Handle new rig assignment
	//
	case rpcproto.Method_RigAssignment:

		request := packet.GetRigAssignmentRequest()
		if request == nil {
			ctl.log.WithFields(logrus.Fields{
				"method": packet.Method.String(),
				"params": "RigAssignmentRequest",
			}).Error("Params are nil")
			return errors.New("params are nil")
		}

		ctl.log.WithFields(logrus.Fields{
			"method": packet.Method.String(),
			"params": "RigAssignmentRequest",
		}).Debug("New RPC message processing")

		err = ctl.handleAssignment(request)
		if err != nil {
			ctl.log.Errorf("Unable to update mining assignment: %s", err)
			// Send response message
			response := rpcproto.Packet{
				Method: rpcproto.Method_RigAssignment,
				Params: &rpcproto.Packet_RigAssignmentResponse{
					RigAssignmentResponse: &rpcproto.RigAssignmentResponse{
						Status:     "RigAssignment error",
						StatusCode: http.StatusInternalServerError,
						Reason:     fmt.Sprintf("Unable to update rig assignment: %s", err),
					},
				},
			}
			err = ctl.sendMessage(&response)
			if err != nil {
				ctl.log.Errorf("Unable to send RigAssignmentResponse to MiningHQ: %s", err)
			}
			return err
		}
		ctl.log.Info("Rig has been reconfigured with new mining assignment")

		// Send response message
		response := rpcproto.Packet{
			Method: rpcproto.Method_RigAssignment,
			Params: &rpcproto.Packet_RigAssignmentResponse{
				RigAssignmentResponse: &rpcproto.RigAssignmentResponse{
					Status:     "Ok",
					StatusCode: http.StatusOK,
				},
			},
		}
		err = ctl.sendMessage(&response)
		if err != nil {
			ctl.log.Errorf("Unable to send RigAssignmentResponse to MiningHQ: %s", err)
		}

	//
	// Handle rig information received
	//
	case rpcproto.Method_RigInfo:
		request := packet.GetRigInfoResponse()
		if request == nil {
			ctl.log.WithFields(logrus.Fields{
				"method": packet.Method.String(),
				"params": "RigInfoResponse",
			}).Error("Params are nil")
			return errors.New("params are nil")
		}

		ctl.log.WithFields(logrus.Fields{
			"method": packet.Method.String(),
			"params": "RigInfoResponse",
		}).Debug("New RPC message processing")

		ctl.currentInfo = request
		ctl.log.WithFields(logrus.Fields{
			"method": packet.Method.String(),
			"params": "RigInfoResponse",
			"name":   ctl.currentInfo.Name,
			"link":   ctl.currentInfo.Link,
		}).Info("Updated local rig information")

		//
		// Handle incoming account stats
		//
	case rpcproto.Method_Stats:
		// If we receive account stats it means that
		// the discord integration is enabled and we need to push this
		// update to Discord
		response := packet.GetStatsResponse()
		if response == nil {
			ctl.log.WithFields(logrus.Fields{
				"method": packet.Method.String(),
				"params": "StatsResponse",
			}).Error("Params are nil")
			// No stats, clear the presence
			ctl.clearDiscordPresence()
			return errors.New("params are nil")
		}

		ctl.log.WithFields(logrus.Fields{
			"method": packet.Method.String(),
			"params": "StatsResponse",
		}).Debug("New RPC message processing")

		if len(response.Stats) == 0 {
			ctl.log.WithFields(logrus.Fields{
				"method": packet.Method.String(),
				"params": "StatsResponse",
			}).Error("No stats received from MiningHQ")
			// No stats, clear the presence
			ctl.clearDiscordPresence()
			return errors.New("no stats received")
		}

		// If the account hashrate is zero, clear the presence
		if response.Stats[0].Hashrate <= 0.00 {
			ctl.clearDiscordPresence()
			return nil
		}
		// Set the Discord rich presence
		ctl.setDiscordHashrate(response.Stats[0].Hashrate)

	default:
		ctl.log.WithField(
			"method", packet.Method,
		).Warning("Unknown method request received")
	}
	return nil
}

// sendMessage takes a Packet protocol, serializes it
// and sends it to MiningHQ over websocket
func (ctl *Ctl) sendMessage(packet *rpcproto.Packet) error {
	packetBytes, err := proto.Marshal(packet)
	if err != nil {
		return err
	}
	return ctl.client.WriteMessage(packetBytes)
}

// trackAndSubmitStats gets the stats from the miners and submits it
// periodically to MiningHQ
func (ctl *Ctl) trackAndSubmitStats() {

	for {
		var err error
		var packet rpcproto.Packet

		ctl.mutex.Lock()
		minerCount := len(ctl.miners)
		ctl.mutex.Unlock()
		// If we have no miners and not in the mining state, then stop sending stats
		if minerCount == 0 || ctl.currentState != rpcproto.MinerState_Mining {
			ctl.log.Debug("No miners connected or not mining, stopping stats")
			return
		}

		statsCollection := ctl.getMinersStats()

		ctl.log.WithFields(logrus.Fields{
			"rig_id": ctl.rigID,
		}).Debug("Sending stats")

		packet = rpcproto.Packet{
			Method: rpcproto.Method_Stats,
			Params: &rpcproto.Packet_StatsResponse{
				StatsResponse: &rpcproto.StatsResponse{
					Stats: statsCollection,
				},
			},
		}
		err = ctl.sendMessage(&packet)
		if err != nil {
			ctl.log.WithField(
				"rig_id", ctl.rigID,
			).Warningf("Unable to send rig stats: %s", err)
			continue
		}

		ctl.log.WithFields(logrus.Fields{
			"rig_id": ctl.rigID,
		}).Debug("Stats sent")

		// TODO: Sleep time for stats config
		time.Sleep(time.Second * 10)

	}
}

// minerErrorHandler handles errors reported by the miner
func (ctl *Ctl) minerErrorHandler(minerKey string, errorText string) {
	ctl.log.WithFields(logrus.Fields{
		"key": minerKey,
	}).Errorf("Detected miner error: %s", errorText)

	var packet rpcproto.Packet
	// If the output contains 'error', generate and error, otherwise a warning
	if strings.Contains(strings.ToLower(errorText), "error") {
		packet = rpcproto.Packet{
			Method: rpcproto.Method_RigError,
			Params: &rpcproto.Packet_RigError{
				RigError: &rpcproto.RigErrorDetail{
					MinerKey: minerKey,
					Reason:   errorText,
				},
			},
		}
	} else {
		packet = rpcproto.Packet{
			Method: rpcproto.Method_RigWarning,
			Params: &rpcproto.Packet_RigWarning{
				RigWarning: &rpcproto.RigWarningDetail{
					MinerKey: minerKey,
					Reason:   errorText,
				},
			},
		}
	}

	err := ctl.sendMessage(&packet)
	if err != nil {
		ctl.log.Errorf(
			"Unable to send RigAssignmentResponse to MiningHQ: %s",
			err)
	}
}

// GetInfo returns the information about the rig
func (ctl *Ctl) GetInfo(
	ctx context.Context,
	request *rpcproto.RigInfoRequest) (*rpcproto.RigInfoResponse, error) {

	ctl.log.WithFields(logrus.Fields{
		"method": "GetInfo",
	}).Debug("New gRPC message processing")

	if ctl.currentInfo == nil {
		return nil, errors.New("No info received from MiningHQ yet")
	}

	return ctl.currentInfo, nil
}

// GetState requests the current rig state
func (ctl *Ctl) GetState(
	ctx context.Context,
	request *rpcproto.StateRequest) (*rpcproto.StateResponse, error) {

	ctl.log.WithFields(logrus.Fields{
		"method": "GetState",
	}).Debug("New gRPC message processing")

	response := rpcproto.StateResponse{
		State:      ctl.currentState,
		Status:     "Ok",
		StatusCode: http.StatusOK,
	}

	return &response, nil
}

// SetState requests the rig to enter the specified state
func (ctl *Ctl) SetState(
	ctx context.Context,
	request *rpcproto.StateRequest) (*rpcproto.StateResponse, error) {

	ctl.log.WithFields(logrus.Fields{
		"method": "SetState",
	}).Debug("New gRPC message processing")

	err := ctl.handleControl(request)
	if err != nil {
		ctl.log.WithField(
			"method", "SetState",
		).Errorf("Unable to update state: %s", err)

		response := rpcproto.StateResponse{
			Status:     "StateResponse error",
			StatusCode: http.StatusInternalServerError,
			Reason:     fmt.Sprintf("Unable to update rig state: %s", err),
		}

		return &response, err
	}
	ctl.log.Info("Rig state has been updated")

	// Send response message
	response := rpcproto.Packet{
		Method: rpcproto.Method_State,
		Params: &rpcproto.Packet_StateResponse{
			StateResponse: &rpcproto.StateResponse{
				Status:     "Ok",
				StatusCode: http.StatusOK,
				State:      ctl.currentState,
			},
		},
	}
	err = ctl.sendMessage(&response)
	if err != nil {
		ctl.log.Errorf("Unable to send StateResponse to MiningHQ: %s", err)
	}

	return &rpcproto.StateResponse{
		Status:     "Ok",
		StatusCode: http.StatusOK,
		State:      ctl.currentState,
	}, nil
}

// GetStats requests the current stats from the rig
func (ctl *Ctl) GetStats(
	ctx context.Context,
	request *rpcproto.StatsRequest) (*rpcproto.StatsResponse, error) {

	ctl.log.WithFields(logrus.Fields{
		"method": "GetStats",
	}).Debug("New gRPC message processing")

	statsCollection := ctl.getMinersStats()

	response := rpcproto.StatsResponse{
		Stats: statsCollection,
	}

	ctl.log.WithFields(logrus.Fields{
		"method": "GetStats",
		"rig_id": ctl.rigID,
	}).Debug("Stats returned")

	return &response, nil
}

// GetLogs requests a rig's logs
func (ctl *Ctl) GetLogs(
	cts context.Context,
	request *rpcproto.LogsRequest) (*rpcproto.LogsResponse, error) {

	ctl.log.WithFields(logrus.Fields{
		"method": "GetLogs",
	}).Debug("New gRPC message processing")

	response := rpcproto.LogsResponse{
		MinerLogs: ctl.getMinersLogs(),
	}

	ctl.log.WithFields(logrus.Fields{
		"method": "GetLogs",
		"rig_id": ctl.rigID,
	}).Debug("Logs returned")

	return &response, nil
}

// Stop the core controller
func (ctl *Ctl) Stop() error {
	defer ctl.log.Info("Shutdown")

	// Stop the gRPC Manager API
	ctl.grpcServer.Stop()

	// We need to stop all the miners
	ctl.mutex.Lock()
	defer ctl.mutex.Unlock()
	for _, miner := range ctl.miners {
		miner.Stop()
	}
	ctl.miners = nil
	ctl.currentState = rpcproto.MinerState_StopMining
	ctl.clearDiscordPresence()
	return ctl.client.Stop()
}

// getMinersStats retrieves the stats from each active miner and returns
// the slice with all the stats
func (ctl *Ctl) getMinersStats() []*rpcproto.MinerStats {
	var statsCollection []*rpcproto.MinerStats

	ctl.mutex.Lock()
	// Collect stats for all the miners
	for _, miner := range ctl.miners {
		var stats rpcproto.MinerStats
		stats, err := miner.GetStats()
		if err != nil {
			ctl.log.WithField(
				"rig_id", ctl.rigID,
			).Warningf("Unable to read miner (%s) stats: %s", miner.GetType(), err)
			continue
		}

		minerStats := stats
		statsCollection = append(statsCollection, &minerStats)

		ctl.log.WithFields(logrus.Fields{
			"rig_id":   ctl.rigID,
			"miner":    fmt.Sprintf("%s (%s)", miner.GetType(), miner.GetKey()),
			"hashrate": stats.Hashrate,
		}).Debug("Collected stats")
	}
	ctl.mutex.Unlock()
	return statsCollection
}

// getMinersLogs retrieves the logs from each active miner and returns
// the slice with all the logs
func (ctl *Ctl) getMinersLogs() []*rpcproto.MinerLog {
	var logs []*rpcproto.MinerLog
	ctl.mutex.Lock()
	for _, miner := range ctl.miners {
		minerLogs := rpcproto.MinerLog{
			Key:  miner.GetKey(),
			Logs: miner.GetLogs(),
		}
		logs = append(logs, &minerLogs)
	}
	ctl.mutex.Unlock()
	return logs
}

// humanizeHashrate takes an integer hashrate and returns the KH or MH equivalent
func (ctl *Ctl) humanizeHashrate(hashrate float64) string {
	if hashrate > 1000000 {
		return fmt.Sprintf("%.2f MH/s", hashrate/1000000)
	} else if hashrate > 1000 {
		return fmt.Sprintf("%.2f KH/s", hashrate/1000)
	}
	return fmt.Sprintf("%.2f H/s", hashrate)
}

// setDiscordHashrate sets the hashrate in Discord for the current user
func (ctl *Ctl) setDiscordHashrate(hashrate float64) {
	client.Login("530821687864983554")
	client.SetActivity(client.Activity{
		State:   fmt.Sprintf("at %s", ctl.humanizeHashrate(hashrate)),
		Details: "Mining cryptocurrency",
		Assets: client.Assets{
			LargeImage: "Unknown",   // TODO: Add image
			LargeText:  "None",      // TODO: Add image alt
			SmallImage: "Unkown",    // TODO: Add image
			SmallText:  "NoneSmall", // TODO: Add image alt
		},
	})
	ctl.log.Debug("Updated Discord presence")
}

// clearDiscordPresence clears the current set presence in Discord
func (ctl *Ctl) clearDiscordPresence() {
	client.Logout()
}

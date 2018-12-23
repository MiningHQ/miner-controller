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
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/donovansolms/mininghq-miner-controller/src/mhq"
	"github.com/donovansolms/mininghq-miner-controller/src/miner"
	"github.com/donovansolms/mininghq-rpcproto/rpcproto"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Ctl implements the core command and control functionality to communicate
// with MiningHQ and manage the miners on the local rig
type Ctl struct {
	mutex             sync.Mutex
	rigID             string
	websocketEndpoint string
	miningKey         string
	// miners hold the current active miners
	miners       []miner.Miner
	currentState rpcproto.MinerState
	client       *mhq.WebSocketClient
	log          *logrus.Entry
}

// New creates a new instance of the core controller
func New(
	websocketEndpoint string,
	miningKey string,
	rigID string,
	log *logrus.Entry,
) (*Ctl, error) {

	ctl := Ctl{
		rigID:             rigID,
		websocketEndpoint: websocketEndpoint,
		miningKey:         miningKey,
		log:               log,
	}

	return &ctl, nil
}

// Run the core controller
func (ctl *Ctl) Run() error {
	ctl.log.Info("Started")

	var err error
	// retry forever to connect
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

	// Once the current rig specs have been processed by MiningHQ, we'll
	// receive the RigAssignment and start mining
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
		//ctl.Stop()
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
	// Handle incoming errors
	//
	case rpcproto.Method_Error:
		// TODO: Implement errors

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

		logsResponse := rpcproto.LogsResponse{}

		ctl.mutex.Lock()
		for _, miner := range ctl.miners {
			minerLogs := rpcproto.MinerLog{
				Key:  miner.GetKey(),
				Logs: miner.GetLogs(),
			}
			logsResponse.MinerLogs = append(logsResponse.MinerLogs, &minerLogs)
		}
		ctl.mutex.Unlock()

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
	default:
		ctl.log.WithField(
			"method", packet.Method,
		).Warning("Unknown method request received")
	}
	return nil
}

// sendMessage takes a WSPacket protocol buffer packet, serializes it
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

	// TODO: Find a way to shut this down
	for {

		var err error
		var packet rpcproto.Packet
		var statsCollection []*rpcproto.MinerStats

		ctl.mutex.Lock()
		// If we have no miners and not in the mining state, the wait
		if len(ctl.miners) == 0 || ctl.currentState != rpcproto.MinerState_Mining {
			ctl.mutex.Unlock()
			ctl.log.Debug("No miners connected or not mining, stopping stats")
			return
		}

		for _, miner := range ctl.miners {
			var stats rpcproto.MinerStats
			stats, err = miner.GetStats()
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

			// HACK TODO: Print logs as a test
			logs := miner.GetLogs()
			fmt.Println(logs)
		}
		ctl.mutex.Unlock()

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

// Stop the core controller
func (ctl *Ctl) Stop() error {
	defer ctl.log.Info("Shutdown")

	// We need to stop all the miners
	ctl.mutex.Lock()
	defer ctl.mutex.Unlock()
	for _, miner := range ctl.miners {
		miner.Stop()
	}
	ctl.miners = nil
	ctl.currentState = rpcproto.MinerState_StopMining
	return ctl.client.Stop()
}

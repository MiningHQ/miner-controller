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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/donovansolms/mininghq-miner-controller/src/mhq"
	"github.com/donovansolms/mininghq-miner-controller/src/miner"
	"github.com/donovansolms/mininghq-spec/spec"
	"github.com/donovansolms/mininghq-spec/spec/caps"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Ctl implements the core command and control functionality to communicate
// with MiningHQ and manage the miners on the local rig
type Ctl struct {
	mutex sync.Mutex
	rigID string
	// miners hold the current active miners
	miners       []miner.Miner
	currentState spec.State
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
		rigID: rigID,
		log:   log,
	}

	var err error
	ctl.log.Debug("Connecting to MiningHQ services")
	// NewWebSocketClient connects to the given endpoint and authenticates
	ctl.client, err = mhq.NewWebSocketClient(
		websocketEndpoint,
		miningKey,
		rigID,
		ctl.onMessage)
	return &ctl, err
}

// Run the core controller
func (ctl *Ctl) Run() error {
	ctl.log.Info("Started")

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

	// Start loop for checking stats
	go func() {
		ctl.trackAndSubmitStats()
	}()

	// Send current rig specs to MiningHQ
	systemInfo, err := caps.GetSystemInfo()
	if err != nil {
		return err
	}
	packet := spec.WSPacket{
		Message: &spec.WSPacket_SystemInfo{
			SystemInfo: systemInfo,
		},
	}
	ctl.sendMessage(&packet)
	// Once the current rig specs have been processed by MiningHQ, we'll
	// receive the RigAssignment and start mining
	err = ctl.client.Start()
	if err != nil {
		switch typedErr := err.(type) {
		case *websocket.CloseError:
			if typedErr.Code != websocket.CloseNormalClosure {
				return err
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
		ctl.Stop()
		return err
	}

	var packet spec.WSPacket
	err = proto.Unmarshal(data, &packet)
	if err != nil {
		ctl.log.Warningf("Unable to process message: %s", err)
		return err
	}

	switch message := packet.Message.(type) {
	case *spec.WSPacket_Error:
		ctl.log.WithField(
			"type", "WSPacket_Error",
		).Debug("New message received")
		_ = message

	case *spec.WSPacket_StateRequest:
		ctl.log.WithField(
			"type", "WSPacket_StateRequest",
		).Debug("New message received")

		err = ctl.handleControl(message.StateRequest)
		if err != nil {
			ctl.log.Errorf("Unable to update state: %s", err)
			// Send response message
			packet := spec.WSPacket{
				Message: &spec.WSPacket_StateResponse{
					StateResponse: &spec.StateResponse{
						Status:     "StateResponse error",
						StatusCode: http.StatusInternalServerError,
						Reason:     fmt.Sprintf("Unable to update rig state: %s", err),
					},
				},
			}
			err = ctl.sendMessage(&packet)
			if err != nil {
				ctl.log.Errorf("Unable to send StateResponse to MiningHQ: %s", err)
			}
		}
		ctl.log.Info("Rig state has been updated")

		// Send response message
		packet := spec.WSPacket{
			Message: &spec.WSPacket_StateResponse{
				StateResponse: &spec.StateResponse{
					Status:     "Ok",
					StatusCode: http.StatusOK,
				},
			},
		}
		err = ctl.sendMessage(&packet)
		if err != nil {
			ctl.log.Errorf("Unable to send StateResponse to MiningHQ: %s", err)
		}

	case *spec.WSPacket_RigAssignment:
		ctl.log.WithField(
			"type", "WSPacket_RigAssignment",
		).Debug("New message received")

		err = ctl.handleAssignment(message.RigAssignment)
		if err != nil {
			ctl.log.Errorf("Unable to update mining assignment: %s", err)
			// Send response message
			packet := spec.WSPacket{
				Message: &spec.WSPacket_RigAssignmentResponse{
					RigAssignmentResponse: &spec.RigAssignmentResponse{
						Status:     "RigAssignment error",
						StatusCode: http.StatusInternalServerError,
						Reason:     fmt.Sprintf("Unable to update rig assignment: %s", err),
					},
				},
			}
			err = ctl.sendMessage(&packet)
			if err != nil {
				ctl.log.Errorf("Unable to send RigAssignmentResponse to MiningHQ: %s", err)
			}
		}
		ctl.log.Info("Rig has been reconfigured with new mining assignment")

		// Send response message
		packet := spec.WSPacket{
			Message: &spec.WSPacket_RigAssignmentResponse{
				RigAssignmentResponse: &spec.RigAssignmentResponse{
					Status:     "Ok",
					StatusCode: http.StatusOK,
				},
			},
		}
		err = ctl.sendMessage(&packet)
		if err != nil {
			ctl.log.Errorf("Unable to send RigAssignmentResponse to MiningHQ: %s", err)
		}

	default:
		ctl.log.WithField(
			"type", "Unknown",
		).Warning("New message received")
	}
	return nil
}

// sendMessage takes a WSPacket protocol buffer packet, serializes it
// and sends it to MiningHQ over websocket
func (ctl *Ctl) sendMessage(packet *spec.WSPacket) error {
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

		ctl.mutex.Lock()
		// If we have no miners and not in the mining state, the wait
		if len(ctl.miners) == 0 || ctl.currentState != spec.State_Mining {
			ctl.mutex.Unlock()
			goto sleep
		}

		for _, miner := range ctl.miners {
			stats, err := miner.GetStats()
			if err != nil {
				ctl.log.WithField(
					"rig_id", ctl.rigID,
				).Warning("Unable to read miner (%s) stats: %s", miner.GetType(), err)
				continue
			}

			ctl.log.WithFields(logrus.Fields{
				"rig_id":   ctl.rigID,
				"hashrate": stats.Hashrate,
			}).Debug("Sending stats")
			packet := &spec.WSPacket{
				Message: &spec.WSPacket_MinerStats{
					MinerStats: &stats,
				},
			}
			err = ctl.sendMessage(packet)
			if err != nil {
				ctl.log.WithField(
					"rig_id", ctl.rigID,
				).Warning("Unable to send miner (%s) stats: %s", miner.GetType(), err)
				continue
			}

			ctl.log.WithFields(logrus.Fields{
				"rig_id":   ctl.rigID,
				"hashrate": stats.Hashrate,
			}).Debug("Stats sent")
		}
		ctl.mutex.Unlock()

	sleep:

		time.Sleep(time.Second * 5)

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
	return ctl.client.Stop()
}

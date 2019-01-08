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
	"os"
	"path/filepath"
	"strconv"

	"github.com/donovansolms/mininghq-miner-controller/src/miner"
	"github.com/donovansolms/mininghq-rpcproto/rpcproto"
	"github.com/sirupsen/logrus"
)

// handleAssignment handles new mining assignments from MiningHQ
func (ctl *Ctl) handleAssignment(assignment *rpcproto.RigAssignmentRequest) error {
	ctl.mutex.Lock()
	defer ctl.mutex.Unlock()

	var err error
	ctl.log.Info("Received new rig assignment")
	// If we were mining, we need to stop all the miners and remove their
	// config files
	ctl.log.Debug("Stopping all miners...")
	for _, miner := range ctl.miners {
		err = miner.Stop()
		if err != nil {
			return fmt.Errorf("Unable to stop miner (%s): %s", miner.GetType(), err)
		}
	}
	ctl.miners = nil

	for i, config := range assignment.MinerConfigs {
		ctl.log.WithFields(logrus.Fields{
			"id": i,
		}).Debug("Configuring miner")

		// TODO / NOTE: go-unattended needs an update when multiple processes attempt to
		// update the same target. Unattended was never *meant* to be run this way
		// but it works very well regardless. For now we only limit a single miner
		// to check for updates.
		// TODO BUG: This has the side-effect of only one miner updating and
		// restarting. The others will only be updated when they are restarted
		// for whatever reason
		//
		// Only the first miner will check for updates
		withUpdate := false
		if i == 0 {
			withUpdate = true
		}

		// TODO: Use the config.Miner specified
		// TODO: Until we support more miners, we'll hardcode xmrig :)

		// TODO: Change API port for each miner!
		// TODO: Write miners and configs to the real dirs

		executablePath, err := os.Executable()
		if err != nil {
			ctl.log.Fatalf("Unable to get current executable path: %s", err)
		}

		// Walk up the tree to determine the correct path
		minerDir := filepath.Dir(executablePath)
		minerDir = filepath.Dir(minerDir)
		minerDir = filepath.Join(minerDir, "miners")

		// Configure miners with new assignment
		xmrig, err := miner.NewXmrig(
			withUpdate,
			filepath.Join(minerDir, "xmrig"),
			filepath.Join(minerDir, "config."+strconv.Itoa(i)+".json"),
			*config,
		)
		if err != nil {
			return fmt.Errorf("Unable to create new miner (xmrig): %s", err)
		}
		xmrig.SetErrorHandler(ctl.minerErrorHandler)

		ctl.miners = append(ctl.miners, xmrig)

		// Start mining again
		ctl.log.WithField(
			"id", i,
		).Debug("Starting miner with new assignment")
		go func(id int) {
			err := xmrig.Start()
			if err != nil {
				ctl.log.WithField(
					"id", id,
				).Errorf("Unable to start miner: %s", err)

				packet := rpcproto.Packet{
					Method: rpcproto.Method_RigError,
					Params: &rpcproto.Packet_RigError{
						RigError: &rpcproto.RigErrorDetail{
							MinerKey: config.GetKey(),
							Reason:   fmt.Sprintf("Unable to start rig miner: %s", err),
						},
					},
				}
				err = ctl.sendMessage(&packet)
				if err != nil {
					ctl.log.Errorf(
						"Unable to send RigAssignmentResponse to MiningHQ: %s",
						err)
				}
			}
		}(i)

		ctl.currentState = rpcproto.MinerState_Mining
	}
	// Start loop for checking stats
	if ctl.currentState == rpcproto.MinerState_Mining {
		go func() {
			ctl.trackAndSubmitStats()
		}()
	}
	ctl.currentAssignment = assignment
	return nil
}

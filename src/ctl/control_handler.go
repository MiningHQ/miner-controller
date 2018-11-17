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

	"github.com/donovansolms/mininghq-spec/spec"
)

// handleControl handles new control messages from MiningHQ
func (ctl *Ctl) handleControl(control *spec.StateRequest) error {
	ctl.mutex.Lock()
	defer ctl.mutex.Unlock()

	var err error
	if control.State == spec.State_StopMining {
		ctl.log.WithField(
			"state", spec.State_StopMining.String(),
		).Info("Received new control state")
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
		ctl.currentState = spec.State_StopMining
	}
	return nil
}

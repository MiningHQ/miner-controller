package ctl

import (
	"fmt"
	"strconv"

	"github.com/donovansolms/mininghq-miner-controller/src/miner"
	"github.com/donovansolms/mininghq-spec/spec"
	"github.com/sirupsen/logrus"
)

// handleAssignment handles new mining assignments from MiningHQ
func (ctl *Ctl) handleAssignment(assignment *spec.RigAssignment) error {
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
		// TODO: The assignment should (probably) determine the miner to use
		// TODO: Until we support more miners, we'll hardcode xmrig :)

		// TODO: Change API port for each miner!
		//
		// Configure miners with new assignment
		xmrig, err := miner.NewXmrig(
			withUpdate,
			"/tmp/miners/xmrig",
			"/tmp/config"+strconv.Itoa(i)+".json",
			*config,
		)
		if err != nil {
			return fmt.Errorf("Unable to create new miner (xmrig): %s", err)
		}

		ctl.miners = append(ctl.miners, xmrig)

		// Start mining again
		ctl.log.WithField(
			"id", i,
		).Debug("Starting miner with new assignment")
		go func() {
			err := xmrig.Start()
			if err != nil {
				// TODO We should send a message back to MiningHQ when we
				// can't start the miner
				//panic(err)
				fmt.Println("err:", err)
			}
		}()
	}
	return nil
}

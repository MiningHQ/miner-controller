package ctl

import (
	"fmt"

	"github.com/donovansolms/mininghq-miner-controller/src/miner"
)

// Ctl implements the core command and control functionality to communicate
// with MiningHQ and manage the miners on the local rig
type Ctl struct {
	miners []miner.Miner
}

// New creates a new instance of the core controller
func New() (*Ctl, error) {
	ctl := Ctl{}

	return &ctl, nil
}

// Run the core controller
func (ctl *Ctl) Run() error {
	fmt.Println("Run controller!")

	// TODO: Connect and authenticate to the MiningHQ service

	return nil
}

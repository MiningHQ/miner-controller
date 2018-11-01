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

package miner

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	unattended "github.com/ProjectLimitless/go-unattended"
	"github.com/donovansolms/mininghq-spec/spec"
	"github.com/sirupsen/logrus"
)

// Xmrig implements the miner interface for xmrig
type Xmrig struct {
	// configPath might differ from the miner's location due to
	// how MiningHQ's split mining is implemented
	configPath    string
	withUpdate    bool
	updateWrapper *unattended.Unattended
}

type xmrigPool struct {
	URL            string      `json:"url"`
	User           string      `json:"user"`
	Pass           string      `json:"pass"`
	RigID          string      `json:"rig-id"`
	Nicehash       bool        `json:"nicehash"`
	Keepalive      bool        `json:"keepalive"`
	Variant        string      `json:"variant"`
	TLS            bool        `json:"tls"`
	TLSFingerprint interface{} `json:"tls-fingerprint"`
}

// cpuConfigSpec contains the options to write to the xmrig JSON config
type xmrigCPUConfigSpec struct {
	API struct {
		Port       int  `json:"port"`
		Ipv6       bool `json:"ipv6"`
		Restricted bool `json:"restricted"`
	} `json:"api"`
	Asm         string      `json:"asm"`
	Autosave    bool        `json:"autosave"`
	Av          int         `json:"av"`
	Background  bool        `json:"background"`
	Colors      bool        `json:"colors"`
	DonateLevel int         `json:"donate-level"`
	HugePages   bool        `json:"huge-pages"`
	HwAes       bool        `json:"hw-aes"`
	Algo        string      `json:"algo"`
	Pools       []xmrigPool `json:"pools"`
	PrintTime   int         `json:"print-time"`
	Retries     int         `json:"retries"`
	RetryPause  int         `json:"retry-pause"`
	Safe        bool        `json:"safe"`
	Threads     int         `json:"threads"`
	UserAgent   string      `json:"user-agent"`
	Syslog      bool        `json:"syslog"`
	Watch       bool        `json:"watch"`
}

// NewXmrig creates a new instance of the xmrig CPU miner
//
// It takes the unattended base path, the path to use for the config
// and the configuration to use
//
// We configure the miner at construction
func NewXmrig(
	withUpdate bool,
	basePath string,
	configPath string,
	config spec.MinerConfig) (*Xmrig, error) {
	xmrig := Xmrig{
		withUpdate: withUpdate,
		configPath: configPath,
	}

	err := xmrig.configure(config)
	if err != nil {
		return nil, err
	}

	// Setup the logging, by default we log to stdout
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "Jan 02 15:04:05",
	})
	logrus.SetLevel(logrus.InfoLevel)
	// TODO: Debug?
	logrus.SetLevel(logrus.DebugLevel)

	logrus.SetOutput(os.Stdout)
	log := logrus.WithFields(logrus.Fields{
		"service": "unattended",
	})
	log.Info("Setting up Unattended updates")

	xmrig.updateWrapper, err = unattended.New(
		"TEST001", // clientID
		unattended.Target{ // target
			VersionsPath:    basePath,
			AppID:           "xmrig",
			UpdateEndpoint:  "http://unattended-old.local",
			UpdateChannel:   "stable",
			ApplicationName: "xmrig",
			ApplicationParameters: []string{
				"--config",
				configPath,
			},
		},
		time.Hour, // UpdateCheckInterval
		log,
	)
	if err != nil {
		return nil, err
	}

	if xmrig.withUpdate {
		// During construction we check for any updates as well, this has the
		// side effect that *if* the miner doesn't exist yet, it will be downloaded
		_, err = xmrig.updateWrapper.ApplyUpdates()
	}
	return &xmrig, err
}

// configure xmrig via the config file. Once reconfigured, the miner
// would need to be restarted
func (miner *Xmrig) configure(config spec.MinerConfig) error {

	if config.CPUConfig == nil {
		return fmt.Errorf("You must provide a CPUConfig for xmrig")
	}

	cpuConfig := miner.generateDefaultCPUConfig()
	cpuConfig.Threads = int(config.CPUConfig.ThreadCount)
	cpuConfig.Algo = config.Algorithm
	cpuConfig.Pools = []xmrigPool{
		{
			URL:     config.PoolConfig.Endpoint,
			User:    config.PoolConfig.Username,
			Pass:    config.PoolConfig.Password,
			Variant: config.PoolConfig.Variant,
		},
	}
	return miner.writeConfig(cpuConfig)
}

// Start xmrig
func (miner *Xmrig) Start() error {
	if miner.withUpdate {
		//Check for and apply updates first
		miner.updateWrapper.ApplyUpdates()
		return miner.updateWrapper.Run()
	}
	return miner.updateWrapper.RunWithoutUpdate()
}

// Stop the miner and remove the config files
func (miner *Xmrig) Stop() error {
	err := miner.updateWrapper.Stop()
	if err != nil {
		return err
	}
	return os.Remove(miner.configPath)
}

// GetType returns the miner type
func (miner *Xmrig) GetType() string {
	return "xmrig"
}

// GetStats returns the mining stats in a uniform format from xmrig
func (miner *Xmrig) GetStats() error {
	return nil
}

// writeConfig writes the config to the drive
func (miner *Xmrig) writeConfig(config xmrigCPUConfigSpec) error {
	configFile, err := os.OpenFile(
		miner.configPath,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0644)
	if err != nil {
		return err
	}
	err = json.NewEncoder(configFile).Encode(config)
	if err != nil {
		return err
	}

	return nil
}

// generateDefaultCPUConfig creates a config with some sane defaults
func (miner *Xmrig) generateDefaultCPUConfig() xmrigCPUConfigSpec {
	config := xmrigCPUConfigSpec{}
	// TODO: API port needs to be different for each miner... duh
	// maybe freeport can help. It would need to return the port
	config.API.Port = 5000
	config.API.Ipv6 = false
	config.API.Restricted = true
	config.Asm = "auto"
	config.Autosave = false
	config.Av = 0
	// TODO: update this to hide the miner
	config.Background = false
	// TODO: Color for now
	config.Colors = true
	// TODO: determined by account
	config.DonateLevel = 5
	config.HugePages = true
	// TODO: Check if we pass this in the CPU config
	config.HwAes = true
	config.Algo = "cryptonight"
	config.PrintTime = 60
	config.Retries = 5
	config.RetryPause = 5
	config.Safe = false
	config.Threads = 1
	config.Syslog = false
	// Watch is currently not supported in xmrig, only in xmrig-proxy
	config.Watch = false
	return config
}

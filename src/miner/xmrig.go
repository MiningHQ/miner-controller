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
	"container/list"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	unattended "github.com/ProjectLimitless/go-unattended"
	"github.com/donovansolms/mininghq-spec/spec"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
)

// Xmrig implements the miner interface for xmrig
type Xmrig struct {
	// configPath might differ from the miner's location due to
	// how MiningHQ's split mining is implemented
	configPath    string
	withUpdate    bool
	updateWrapper *unattended.Unattended

	key      string
	apiPort  int
	logList  *list.List
	logMax   int
	logMutex sync.Mutex
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

// xmrigAPIResponse is returned from the normal miner API
type xmrigAPIResponse struct {
	ID       string `json:"id"`
	WorkerID string `json:"worker_id"`
	Version  string `json:"version"`
	Kind     string `json:"kind"`
	Ua       string `json:"ua"`
	CPU      struct {
		Brand   string `json:"brand"`
		Aes     bool   `json:"aes"`
		X64     bool   `json:"x64"`
		Sockets int    `json:"sockets"`
	} `json:"cpu"`
	Algo        string `json:"algo"`
	Hugepages   bool   `json:"hugepages"`
	DonateLevel int    `json:"donate_level"`
	Hashrate    struct {
		Total   []float64   `json:"total"`
		Highest float64     `json:"highest"`
		Threads [][]float64 `json:"threads"`
	} `json:"hashrate"`
	Results struct {
		DiffCurrent uint64        `json:"diff_current"`
		SharesGood  uint32        `json:"shares_good"`
		SharesTotal uint32        `json:"shares_total"`
		AvgTime     int           `json:"avg_time"`
		HashesTotal uint64        `json:"hashes_total"`
		Best        []int         `json:"best"`
		ErrorLog    []interface{} `json:"error_log"`
	} `json:"results"`
	Connection struct {
		Pool     string        `json:"pool"`
		Uptime   int           `json:"uptime"`
		Ping     int           `json:"ping"`
		Failures int           `json:"failures"`
		ErrorLog []interface{} `json:"error_log"`
	} `json:"connection"`
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
		key:        config.Key,
		withUpdate: withUpdate,
		configPath: configPath,
		logList:    list.New(),
		logMax:     100,
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

	cpuConfig, err := miner.generateDefaultCPUConfig()
	if err != nil {
		return fmt.Errorf("unable to create config: %s", err)
	}
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
	// Setup the reading of the output
	// TODO: Add back log handling
	//outputReader, outputWriter := io.Pipe()
	//miner.updateWrapper.SetOutputWriter(outputWriter)
	miner.updateWrapper.SetOutputWriter(os.Stdout)
	// go func() {
	// 	scanner := bufio.NewScanner(outputReader)
	// 	for scanner.Scan() {
	// 		miner.logMutex.Lock()
	// 		miner.logList.PushBack(scanner.Text())
	// 		if miner.logList.Len() >= miner.logMax {
	// 			miner.logList.Remove(miner.logList.Front())
	// 		}
	// 		miner.logMutex.Unlock()
	//
	// 		// TODO: Can this not be done with the API?
	// 		if strings.Contains(strings.ToLower(scanner.Text()), "error") {
	// 			fmt.Println("\n\nDETERTTED ERROR: ", scanner.Text())
	// 		}
	// 	}
	// }()
	//

	// // HACK / TEST
	// go func() {
	// 	for {
	// 		stats, _ := miner.GetStats()
	// 		fmt.Println(stats)
	// 		time.Sleep(time.Second * 5)
	// 	}
	// }()

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
func (miner *Xmrig) GetStats() (spec.MinerStats, error) {

	var stats spec.MinerStats

	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d", miner.apiPort))
	if err != nil {
		fmt.Println(err)
		return stats, err
	}

	var xmrigStats xmrigAPIResponse
	err = json.NewDecoder(response.Body).Decode(&xmrigStats)
	if err != nil {
		fmt.Println(err)
		return stats, err
	}
	stats.Key = miner.key
	stats.Hashrate = xmrigStats.Hashrate.Total[0]
	stats.MaxHashrate = xmrigStats.Hashrate.Highest
	stats.TotalHashes = xmrigStats.Results.HashesTotal
	stats.CurrentDifficulty = xmrigStats.Results.DiffCurrent
	stats.TotalShares = xmrigStats.Results.SharesTotal
	stats.AcceptedShares = xmrigStats.Results.SharesGood
	stats.RejectedShares = stats.TotalShares - stats.AcceptedShares
	return stats, nil
}

// GetLogs returns the last logs from the actual miner
func (miner *Xmrig) GetLogs() []string {
	// Get all the logs and return them in the current order
	miner.logMutex.Lock()
	defer miner.logMutex.Unlock()

	var logs []string
	for item := miner.logList.Front(); item != nil; item = item.Next() {
		log := item.Value.(string)
		logs = append(logs, log)
	}
	return logs
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
func (miner *Xmrig) generateDefaultCPUConfig() (xmrigCPUConfigSpec, error) {
	config := xmrigCPUConfigSpec{}

	port, err := freeport.GetFreePort()
	if err != nil {
		return config, err
	}

	miner.apiPort = port
	config.API.Port = port
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
	return config, nil
}

/*
  MiningHQ Miner Controller - manages cryptocurrency miners on a user's machine.
  https://mininghq.io

	Copyright (C) 2018  Donovan Solms     <https://github.com/donovansolms>
                                        <https://github.com/mininghq>

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

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/mininghq/miner-controller/src/conf"
	"github.com/mininghq/miner-controller/src/ctl"
	logrus "github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
)

// Config holds the environment variables for this service
type Config struct {
	Debug bool `split_words:"true"`
}

func main() {

	var config Config
	logLevel := logrus.InfoLevel
	err := envconfig.Process("", &config)
	if err != nil {
		logrus.Fatal("Unable to process config", err)
	}
	logrus.SetOutput(os.Stdout)

	logOutputFormat := logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "Jan 02 15:04:05",
	}
	logrus.SetFormatter(&logOutputFormat)

	if config.Debug {
		logLevel = logrus.DebugLevel
	}
	logrus.SetLevel(logLevel)
	logger := logrus.WithFields(logrus.Fields{
		"service_class": "miner-controller",
	})

	// grpcEndpoint is the gRPC API endpoint used by the Miner Manager to
	// communicate with the miner controller
	grpcEndpoint := "localhost:64630" // Port = MINE0
	// websocketEndpoint is the websocket endpoint connection of MiningHQ
	// to which we connect for command and control
	websocketEndpoint := conf.WebsocketEndpoint

	// The structure of the folders should be
	// /miner-controller
	// 	/logs
	// 		/mininghq.log
	// 	/{versions}
	// 		/mininghq-miner-controller
	// 	/mining_key
	// 	/rig_id

	// executablePath is the full path to the binary
	// /miner-controller/{version}/mininghq-miner-controller
	// /miner-controller/v0.0.0.1/mininghq-miner-controller
	executablePath, err := os.Executable()
	if err != nil {
		logger.Fatalf("Unable to find executing path: %s", err)
	}
	// basePath is the dir the executable is in
	// /miner-controller/v0.0.0.1
	basePath := filepath.Dir(executablePath)
	// Go up one more
	// /miner-controller
	basePath = filepath.Dir(basePath)

	logsDir := filepath.Join(basePath, "logs")
	// Make the logs dir
	err = os.MkdirAll(logsDir, 0755)
	if err != nil {
		logger.Fatalf("Unable to create logging directory: %s", err)
	}

	rotateFileHook, err := rotatefilehook.NewRotateFileHook(rotatefilehook.RotateFileConfig{
		Filename:   filepath.Join(logsDir, "mininghq.log"),
		MaxSize:    100, // 100MB files will be rolled
		MaxBackups: 3,   // Keep a maximum of 3 logfiles
		MaxAge:     3,   // Keep logfiles for a maximum of 3 days
		// TODO: Add the lumberjack compression
		Level:     logLevel,
		Formatter: &logOutputFormat,
	})
	if err != nil {
		logger.Errorf("Unable to setup file hook: %s", err)
	}
	logrus.AddHook(rotateFileHook)

	// Get the user's mining key that was installed
	miningKey, err := ioutil.ReadFile(filepath.Join(basePath, "mining_key"))
	if err != nil {
		logger.Fatalf("Unable to read rig mining key: %s", err)
	}

	// Get the rig's ID from registration
	rigID, err := ioutil.ReadFile(filepath.Join(basePath, "rig_id"))
	if err != nil {
		logger.Fatalf("Unable to read rig id: %s", err)
	}

	controller, err := ctl.New(
		websocketEndpoint,
		grpcEndpoint,
		strings.TrimSpace(string(miningKey)),
		strings.TrimSpace(string(rigID)),
		logger,
	)
	if err != nil {
		logger.Fatal(err)
	}

	// Run the miner controller
	err = controller.Run()
	if err != nil {
		logger.Fatal(err)
	}
}

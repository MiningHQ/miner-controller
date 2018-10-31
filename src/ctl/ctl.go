package ctl

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/donovansolms/mininghq-miner-controller/src/mhq"
	"github.com/donovansolms/mininghq-miner-controller/src/miner"
	"github.com/gorilla/websocket"
)

// Ctl implements the core command and control functionality to communicate
// with MiningHQ and manage the miners on the local rig
type Ctl struct {
	// miners hold the current active miners
	miners []miner.Miner
	client *mhq.WebSocketClient
}

// New creates a new instance of the core controller
func New(
	websocketEndpoint string,
	miningKey string,
	rigID string,
) (*Ctl, error) {

	ctl := Ctl{}

	var err error
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
	fmt.Println("Run controller!")

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

	// TODO: Send current rig specs to server
	ctl.client.WriteMessage([]byte("ass"))
	// TODO: Server should send back the miner config

	err := ctl.client.Start()
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
func (ctl *Ctl) onMessage(data []byte, err error) {
	fmt.Println("ONMESSAGE")
	fmt.Println(string(data))
}

// Stop the core controller
func (ctl *Ctl) Stop() error {
	fmt.Println("Stop controller!")

	return ctl.client.Stop()
}

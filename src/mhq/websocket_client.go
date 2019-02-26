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

package mhq

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mininghq/miner-controller/src/conf"
)

// WebSocketClient implements a basic websocket client for communicating
// with the MiningHQ service
type WebSocketClient struct {
	sync.Mutex
	// endpoint to connect to
	endpoint string
	// conn is the websocker connection
	conn *websocket.Conn
	// pingTicker triggers the keep alive pings
	pingTicker *time.Ticker
	// onMessage is a callback when a new websocket message is received
	onMessage func([]byte, error) error
}

// NewWebSocketClient creates a new instance of the websocket client
func NewWebSocketClient(
	endpoint string,
	miningKey string,
	rigID string,
	onMessage func([]byte, error) error) (*WebSocketClient, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("The endpoint for WebSocketClient must not be blank")
	}
	if onMessage == nil {
		return nil, fmt.Errorf("You must specify an onMessage callback")
	}
	client := WebSocketClient{
		endpoint:  endpoint,
		onMessage: onMessage,
	}

	headers := http.Header{
		"Authorization": []string{miningKey},
		"X-Rig-ID":      []string{rigID},
	}

	var err error
	var response *http.Response
	client.conn, response, err = websocket.DefaultDialer.Dial(
		client.endpoint,
		headers)
	if err != nil {
		if response != nil {
			if response.StatusCode == http.StatusUnauthorized {
				return nil, fmt.Errorf("Invalid credentials supplied")
			}
		}
		return nil, err
	}
	return &client, nil
}

// Start connects to the endpoint specified and runs the websocket client
func (client *WebSocketClient) Start() error {
	defer client.conn.Close()

	// Only start pinging after connected
	client.pingTicker = time.NewTicker(conf.PingInterval)
	go func() {
		for range client.pingTicker.C {
			client.Ping()
		}
	}()

	err := client.conn.SetReadDeadline(time.Now().Add(conf.PongWait))
	if err != nil {
		return err
	}

	client.conn.SetPongHandler(func(appData string) error {
		err := client.conn.SetReadDeadline(time.Now().Add(conf.PongWait))
		if err != nil {
			return err
		}
		return nil
	})

	for {
		_, data, err := client.conn.ReadMessage()
		err = client.onMessage(data, err)
		if err != nil {
			// switch err.(type) {
			// case *websocket.CloseError:
			// TODO: Find a way to repeatedly check if the MiningHQ websocket is back
			// TODO: Check the error, if connection closed, retry connecting?
			// default:
			// }
			return err
		}
	}
}

// Ping MiningHQ, keeps the connection alive
func (client *WebSocketClient) Ping() error {
	client.Lock()
	defer client.Unlock()
	client.conn.SetWriteDeadline(time.Now().Add(conf.WriteWait))
	return client.conn.WriteMessage(websocket.PingMessage, []byte{0})
}

// WriteMessage send a message via the websocket
func (client *WebSocketClient) WriteMessage(data []byte) error {
	client.Lock()
	defer client.Unlock()
	if client.conn == nil {
		// TODO: Handle
		fmt.Println("NIL CONN!")
	}
	client.conn.SetWriteDeadline(time.Now().Add(conf.WriteWait))
	return client.conn.WriteMessage(websocket.TextMessage, data)
}

// Stop disconnects and closes the websocket connection
func (client *WebSocketClient) Stop() error {
	client.Lock()
	defer client.Unlock()

	// Stop sending pings
	client.pingTicker.Stop()

	// Cleanly close the connection by sending a close message
	err := client.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}

	return nil
}

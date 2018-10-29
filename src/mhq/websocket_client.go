package mhq

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketClient implements a basic websocket client for communicating
// with the MiningHQ service
type WebSocketClient struct {
	sync.Mutex
	// endpoint to connect to
	endpoint string
	conn     *websocket.Conn
	//stopChan chan struct{}

	onMessage func([]byte, error)
}

// NewWebSocketClient creates a new instance of the websocket client
func NewWebSocketClient(
	endpoint string,
	onMessage func([]byte, error)) (*WebSocketClient, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("The endpoint for WebSocketClient must not be blank")
	}
	if onMessage == nil {
		return nil, fmt.Errorf("You must specify an onMessage callback")
	}
	client := WebSocketClient{
		endpoint:  endpoint,
		onMessage: onMessage,
		//stopChan: make(chan struct{}),
	}

	var err error
	client.conn, _, err = websocket.DefaultDialer.Dial(client.endpoint, nil)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

// Start connects to the endpoint specified and runs the websocket client
func (client *WebSocketClient) Start() error {
	defer client.conn.Close()

	//defer close(client.stopChan)
	for {
		_, data, err := client.conn.ReadMessage()
		if err != nil {
			panic(err)
		}
		client.onMessage(data, err)
	}
	return nil
}

// WriteMessage send a message via the websocket
func (client *WebSocketClient) WriteMessage(data []byte) error {
	client.Lock()
	defer client.Unlock()
	fmt.Println("Write!")
	if client.conn == nil {
		fmt.Println("NIL CONN!")
	}
	return client.conn.WriteMessage(websocket.TextMessage, data)
}

// Stop disconnects and closes the websocket connection
func (client *WebSocketClient) Stop() error {

	client.Lock()
	defer client.Unlock()
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := client.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}

	return nil
}

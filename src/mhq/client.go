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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sethgrid/pester"
)

// Client is the API client for MiningHQ APIs
//
// It provides the basic functionality in plain JSON calls. This will be
// moved to a gRPC API in the future
//
// TODO: Implement as gRPC API client
type Client struct {
	// miningKey is the user's mining key. It is used as an identification
	// token for API calls and websocket connections
	miningKey string
	// endpoint of the MiningHQ API
	endpoint string
}

// NewClient creates and returns a new MiningHQ API client
func NewClient(miningKey string, endpoint string) (*Client, error) {
	if miningKey == "" {
		return nil, errors.New(
			"You must provide a valid mining key to the MiningHQ API client, not blank")
	}
	if endpoint == "" {
		return nil, errors.New(
			"You must provide a valid endpoint to the MiningHQ API client, not blank")
	}

	client := Client{
		miningKey: miningKey,
		endpoint:  endpoint,
	}

	return &client, nil
}

// RegisterRig registers this system/rig for the user with MiningHQ
// It returns the MiningHQ RigID
func (client *Client) RegisterRig(
	registerRequest RegisterRigRequest) (string, error) {

	jsonBytes, err := json.Marshal(registerRequest)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/register-rig", client.endpoint),
		bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", client.miningKey))

	apiClient := pester.New()
	apiClient.MaxRetries = 5
	response, err := apiClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("Unable to register rig: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unable to register rig: Status %s", response.Status)
	}

	var registerResponse RegisterRigResponse
	err = json.NewDecoder(response.Body).Decode(&registerResponse)
	if err != nil {
		return "", fmt.Errorf("Unable to read register response: %s", err)
	}

	if registerResponse.Status == "err" {
		return "", errors.New(registerResponse.Message)
	}

	return registerResponse.RigID, nil
}

// DeregisterRig removes this system/rig from the user with MiningHQ
func (client *Client) DeregisterRig(
	deregisterRequest DeregisterRigRequest) error {

	jsonBytes, err := json.Marshal(deregisterRequest)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/deregister-rig", client.endpoint),
		bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", client.miningKey))

	apiClient := pester.New()
	apiClient.MaxRetries = 5
	response, err := apiClient.Do(request)
	if err != nil {
		return fmt.Errorf("Unable to deregister rig: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Unable to deregister rig: Status %s", response.Status)
	}

	var deregisterResponse DeregisterRigResponse
	err = json.NewDecoder(response.Body).Decode(&deregisterResponse)
	if err != nil {
		return fmt.Errorf("Unable to read deregister response: %s", err)
	}

	if deregisterResponse.Status == "err" {
		return errors.New(deregisterResponse.Message)
	}

	return nil
}

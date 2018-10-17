package mhq

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mininghq/miner-controller/src/caps"
	"github.com/sethgrid/pester"
)

// Client is the API client for MiningHQ APIs
//
// It provides the basic functionality in plain JSON calls. This will be
// moved to a gRPC API in the future
//
// TODO: Implement as gRPC API client before moving out of beta
type Client struct {
	// endpoint of the MiningHQ API
	endpoint string
}

// NewClient creates and returns a new MiningHQ API client
func NewClient(endpoint string) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New(
			"You must provide a valid endpoint to the MiningHQ API client")
	}

	client := Client{
		endpoint: endpoint,
	}

	return &client, nil
}

// GetRecommendedMiners submits the rig capabilities to MiningHQ and gets
// back a list of compatible miners, if any
func (client *Client) GetRecommendedMiners(
	systemInfo caps.SystemInfo) ([]RecommendedMiner, error) {

	jsonBytes, err := json.Marshal(systemInfo)
	if err != nil {
		return []RecommendedMiner{}, err
	}

	apiClient := pester.New()
	apiClient.MaxRetries = 5
	response, err := apiClient.Post(
		fmt.Sprintf("%s/recommend-miners", client.endpoint),
		"application/json",
		bytes.NewReader(jsonBytes))
	if err != nil {
		return []RecommendedMiner{},
			fmt.Errorf("Unable to get recommended miners: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return []RecommendedMiner{},
			fmt.Errorf("Unable to get recommended miners: %s", err)
	}

	var recommendedMinerResponse RecommendedMinerResponse
	err = json.NewDecoder(response.Body).Decode(&recommendedMinerResponse)
	if err != nil {
		return []RecommendedMiner{},
			fmt.Errorf("Error reading recommended miners response: %s", err)
	}
	if recommendedMinerResponse.Status == "err" {
		return []RecommendedMiner{}, errors.New(recommendedMinerResponse.Message)
	}

	return recommendedMinerResponse.Miners, nil
}

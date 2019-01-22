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

import "github.com/donovansolms/mininghq-spec/spec/caps"

// Progress holds information about the current download progress
type Progress struct {
	BytesCompleted int64
	BytesTotal     int64
}

// RecommendedMinerResponse contains the recommended miners (if any)
// from the MiningHQ API
type RecommendedMinerResponse struct {
	Status  string             `json:"Status"`
	Message string             `json:"Message"`
	Miners  []RecommendedMiner `json:"Miners"`
}

// RecommendedMiner contains the information to download a recommended miner
type RecommendedMiner struct {
	Name           string `json:"Name"`
	Version        string `json:"Version"`
	Type           string `json:"Type"`
	DownloadLink   string `json:"DownloadLink"`
	DownloadSHA512 string `json:"DownloadSHA512"`
	SizeBytes      int64  `json:"SizeBytes"`
}

// RegisterRigRequest is the request sent to MiningHQ to register a new rig
type RegisterRigRequest struct {
	// Name is a custom name for this rig,
	// if blank, it will be set to the hostname
	Name string
	// Caps is the capabilities of this rig
	Caps *caps.SystemInfo
}

// RegisterRigResponse is returned after a RegisterRigRequest
type RegisterRigResponse struct {
	Status  string `json:"Status"`
	Message string `json:"Message"`
	RigID   string `json:"RigID"`
}

// DeregisterRigRequest is the request sent to MiningHQ to deregister a rig
type DeregisterRigRequest struct {
	// RigID is the identifier for this rig
	RigID string
}

// DeregisterRigResponse is returned after a DeregisterRigRequest
type DeregisterRigResponse struct {
	Status  string `json:"Status"`
	Message string `json:"Message"`
}

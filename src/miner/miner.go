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

package miner

import "github.com/mininghq/rpcproto/rpcproto"

// Miner interface defines the required behaviour for all cryptocurrency miners
type Miner interface {
	// Start the miner
	Start() error
	// Stop the miner and remove the config files
	Stop() error
	// GetType returns the miner type
	GetType() string
	// GetKey returns the miner's config key
	GetKey() string
	// GetStats returns the mining stats in a uniform format
	GetStats() (rpcproto.MinerStats, error)
	// GetLogs returns the last logs from the actual miner
	GetLogs() []string
	// GetVersion returns the latest version currently running
	GetVersion() string
	// SetErrorHandler sets the handler to send any errors to
	// It takes the miner key and the string containing the error
	SetErrorHandler(func(string, string))
}

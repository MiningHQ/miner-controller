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

// Miner interface defines the required behaviour for all cryptocurrency miners
type Miner interface {
	// Start the miner
	Start() error
	// Stop the miner and remove the config files
	Stop() error
	// GetType returns the miner type
	GetType() string
	// GetStats returns the mining stats in a uniform format
	GetStats() error
	// GetLogs returns the last logs from the actual miner
	GetLogs() []string
}

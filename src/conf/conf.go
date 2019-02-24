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

// Package conf contains some base configuration values since we don't include
// any form of configuration file for the service
package conf

import "time"

// Live
// const (
// 	// UnattendedBaseURL is the base URL for the Unattended update service
// 	UnattendedBaseURL = "https://unattended.mininghq.io"
// 	// WebsocketEndpoint is the connection endpoint for websockets. This is used
// 	// to communicate with MiningHQ
// 	WebsocketEndpoint = "ws://www.mininghq.io:9999"
// 	// StatsSubmitInterval defines how long to wait between stats submissions
// 	StatsSubmitInterval = time.Minute
// 	// PingInterval defines how long to wait between sending pings to MiningHQ
//	PingInterval = time.Second * 10
// 	// DiscordAppID is used to submit Discord stats
// 	DiscordAppID = "530821687864983554"
// )

// Dev
const (
	// UnattendedBaseURL is the base URL for the Unattended update service
	UnattendedBaseURL = "http://unattended-old.local"
	// WebsocketEndpoint is the connection endpoint for websockets. This is used
	// to communicate with MiningHQ
	WebsocketEndpoint = "ws://localhost:9999"
	// StatsSubmitInterval defines how long to wait between stats submissions
	StatsSubmitInterval = time.Minute
	// PingInterval defines how long to wait between sending pings to MiningHQ
	PingInterval = time.Second * 10
	// DiscordAppID is used to submit Discord stats
	DiscordAppID = "530821687864983554"
)

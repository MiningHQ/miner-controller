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

package main

import (
	"fmt"

	"github.com/mininghq/miner-controller/src/caps"
	"github.com/mininghq/miner-controller/src/mhq"
)

func main() {
	fmt.Println("Initial commit")

	systemInfo, err := caps.GetSystemInfo()
	if err != nil {
		panic(err)
	}

	mhqClient, err := mhq.NewClient("http://mininghq.local/api")
	if err != nil {
		panic(err)
	}

	recommendedMiners, err := mhqClient.GetRecommendedMiners(systemInfo)
	if err != nil {
		panic(err)
	}

	for i, recommendedMiner := range recommendedMiners {
		fmt.Printf("Recommended miner #%d: %s v%s (%s)\n",
			i,
			recommendedMiner.Name,
			recommendedMiner.Version,
			recommendedMiner.Type)
	}

}

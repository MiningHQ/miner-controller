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
)

func main() {
	fmt.Println("Initial commit")

	memory, err := caps.GetMemoryInfo()
	if err != nil {
		panic(err)
	}
	fmt.Println(memory)

	cpus, err := caps.GetCPUInfo()
	if err != nil {
		panic(err)
	}
	fmt.Println(cpus)

	host, err := caps.GetHostInfo()
	if err != nil {
		panic(err)
	}
	fmt.Println(host)

	gpus, err := caps.GetGPUInfo()
	if err != nil {
		panic(err)
	}
	fmt.Println(gpus)

}

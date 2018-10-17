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
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/donovansolms/mininghq-miner-controller/src/caps"
	"github.com/donovansolms/mininghq-miner-controller/src/mhq"
	pb "gopkg.in/cheggaaa/pb.v1"
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
		fmt.Printf("Downloading miner #%d: %s v%s (%s)\n",
			i,
			recommendedMiner.Name,
			recommendedMiner.Version,
			recommendedMiner.Type)

		// TODO TEMP
		tempFile := fmt.Sprintf("/tmp/miner-%d.tar.gz", time.Now().Unix())
		fmt.Println("Download to", tempFile)
		// progressChan receives progress updates from the selected downloader
		// and is used to display the progress
		progressChan := make(chan mhq.Progress)
		progressBar := pb.New64(recommendedMiner.SizeBytes)
		progressBar.SetUnits(pb.U_BYTES)
		progressBar.Start()

		// We receive the progress via a channel from the downloader
		go func() {
			for progress := range progressChan {
				progressBar.Set64(progress.BytesCompleted)
			}
		}()
		err = mhqClient.DownloadMiner(tempFile, recommendedMiner, progressChan)
		if err != nil {
			fmt.Printf("Download failed: %s\n", err)
			fmt.Print("Press enter to continue...")
			_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
			os.Exit(0)
		}
		// Just in case the progress bar hasn't updated yet, set to 100%
		// since we're done
		progressBar.Set64(recommendedMiner.SizeBytes)
		progressBar.Update()
		progressBar.Finish()

		fmt.Printf("Download saved to %v \n", tempFile)

	}

}

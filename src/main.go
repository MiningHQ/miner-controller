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
	"time"

	"github.com/donovansolms/mininghq-miner-controller/src/mhq"
	"github.com/donovansolms/mininghq-spec/spec"
	"github.com/gogo/protobuf/proto"
)

func main() {

	fmt.Println("Test running the controller")
	//
	// controller, err := ctl.New()
	// if err != nil {
	// 	panic(err)
	// }
	//
	// err = controller.Run()
	// if err != nil {
	// 	panic(err)
	// }

	wsclient, err := mhq.NewWebSocketClient("ws://localhost:9999", func(data []byte, err error) {
		fmt.Println("Got a message!", string(data))
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Start!")
	go func() {
		err = wsclient.Start()
		if err != nil {
			panic(err)
		}
	}()

	msg := spec.LoginRequest{
		MiningKey: "5bd22d231UqU9_vGQwlSP-KX5YIFKi14Gsq_YHEd",
		RigID:     "1i1qWdZ2XSdIrnnvl-BHdFh1kSDQHO6PO",
	}
	packet := spec.WSPacket{
		Message: &spec.WSPacket_LoginRequest{
			LoginRequest: &msg,
		},
	}
	_ = packet

	packetBytes, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}
	_ = packetBytes
	//
	fmt.Println("SEND!!!")
	_ = packetBytes
	err = wsclient.WriteMessage(packetBytes)
	//err = wsclient.WriteMessage([]byte("Test"))
	if err != nil {
		fmt.Println("Error send", err)
	}

	time.Sleep(time.Second * 5)
	fmt.Println("send2")
	err = wsclient.WriteMessage(packetBytes)
	//err = wsclient.WriteMessage([]byte("Test"))
	if err != nil {
		fmt.Println("Error send", err)
	}
	// fmt.Println("SEND!!!")
	// err = wsclient.WriteMessage([]byte("BASTARDS2"))
	// if err != nil {
	// 	fmt.Println("Error send", err)
	// }

	time.Sleep(time.Second * 10)

	//
	// config := spec.MinerConfig{
	// 	Algorithm: "cryptonight",
	// 	PoolConfig: &spec.PoolConfig{
	// 		Endpoint: "mine.stellite.cash:80",
	// 		Username: "Se44JmF1FWQ7ZL6fYNqBu2cHhPvExcvecCKad2kwsdeaCJUE8KjThiRPb6dR4XuXUsad8FsD8DypDC8xpCe85Bfi1wRcdNvS9",
	// 		Password: "test",
	// 		Variant:  "xtl",
	// 	},
	// 	CPUConfig: &spec.CPUConfig{
	// 		ThreadCount: 2,
	// 	},
	// }
	//
	// xmrig, err := miner.NewXmrig(
	// 	"/tmp/miners/xmrig",
	// 	"/tmp/config1.json",
	// 	config,
	// )
	// if err != nil {
	// 	panic(err)
	// }
	//
	// go func() {
	//
	// 	err := xmrig.Start()
	// 	if err != nil {
	// 		//panic(err)
	// 		fmt.Println("err:", err)
	// 	}
	// }()
	//
	// time.Sleep(time.Second * 60)
	//
	// err = xmrig.Stop()
	// if err != nil {
	// 	fmt.Println("Stop err", err)
	// }

	// p := packet.LoginRequest{
	// 	MiningKey: "test",
	// }
	// p.RigID = "XX"
	//
	// t := packet.MinerConfig{
	// 	PoolConfig: &packet.PoolConfig{
	// 		Endpoint: "Seomthing",
	// 		Username: "",
	// 		Password: "",
	// 	},
	// }
	// t.GetCPUConfig()

	//
	// err = beeep.Notify("MiningHQ", "Your miner configuration has been updated", "")
	// if err != nil {
	// 	panic(err)
	// }

	// TODO: Get this key from somewhere2
	//miningKey := "5bd0e44cAb2H4G14n0gz4FEVRyd3Scl0Wgk_UCz6"
	//
	// systemInfo, err := caps.GetSystemInfo()
	// if err != nil {
	// 	panic(err)
	// }
	//
	// fmt.Println(systemInfo)
	//
	// mhqClient, err := mhq.NewClient(miningKey, "http://mininghq.local/api/v1")
	// if err != nil {
	// 	panic(err)
	// }
	//
	// registerRequest := mhq.RegisterRigRequest{
	// 	Name: "testrig",
	// 	Caps: systemInfo,
	// }
	// err = mhqClient.RegisterRig(registerRequest)
	// if err != nil {
	// 	panic(err)
	// }

	//
	// recommendedMiners, err := mhqClient.GetRecommendedMiners(systemInfo)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// for i, recommendedMiner := range recommendedMiners {
	// 	fmt.Printf("Downloading miner #%d: %s v%s (%s)\n",
	// 		i,
	// 		recommendedMiner.Name,
	// 		recommendedMiner.Version,
	// 		recommendedMiner.Type)
	//
	// 	// TODO TEMP
	// 	tempFile := fmt.Sprintf("/tmp/miner-%d.tar.gz", time.Now().Unix())
	// 	fmt.Println("Download to", tempFile)
	// 	// progressChan receives progress updates from the selected downloader
	// 	// and is used to display the progress
	// 	progressChan := make(chan mhq.Progress)
	// 	progressBar := pb.New64(recommendedMiner.SizeBytes)
	// 	progressBar.SetUnits(pb.U_BYTES)
	// 	progressBar.Start()
	//
	// 	// We receive the progress via a channel from the downloader
	// 	go func() {
	// 		for progress := range progressChan {
	// 			progressBar.Set64(progress.BytesCompleted)
	// 		}
	// 	}()
	// 	err = mhqClient.DownloadMiner(tempFile, recommendedMiner, progressChan)
	// 	if err != nil {
	// 		fmt.Printf("Download failed: %s\n", err)
	// 		fmt.Print("Press enter to continue...")
	// 		_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
	// 		os.Exit(0)
	// 	}
	// 	// Just in case the progress bar hasn't updated yet, set to 100%
	// 	// since we're done
	// 	progressBar.Set64(recommendedMiner.SizeBytes)
	// 	progressBar.Update()
	// 	progressBar.Finish()
	//
	// 	fmt.Printf("Download saved to %v \n", tempFile)
	//
	// }

}

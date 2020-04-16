package main

import (
	"github.com/shirou/gopsutil/host"
	"log"
	"time"
)

func (ses *webSocketSession) runSystemStatus() {

	for {
		hostInfo, err := host.Info()
		if err != nil {
			log.Println(err)
			return
		}

		payload := dict{
			"Uptime":          hostInfo.BootTime,
			"Platform":        hostInfo.Platform,
			"PlatformFamily":  hostInfo.PlatformFamily,
			"PlatformVersion": hostInfo.PlatformVersion,
		}

		if err := ses.connection.WriteJSON(payload); err != nil {
			log.Println("WriteJSON Failed:", err)
			break
		}

		time.Sleep(1 * time.Second)
	}

	log.Println("Exiting handleWebSocket")

}

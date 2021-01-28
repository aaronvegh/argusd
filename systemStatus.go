package main

import (
	"github.com/KarpelesLab/strftime"
	"github.com/shirou/gopsutil/host"
	"log"
	"net/http"
	"time"
)

func (app *webSocketApp) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	log.Println("This is /systemStatus starting")

	connection, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	session := &webSocketSession{
		connection: connection,
	}

	session.runSystemStatus()
}

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
			"SystemTime":	   strftime.EnFormat(`%H:%M:%S`, time.Now()),
		}

		if err := ses.connection.WriteJSON(payload); err != nil {
			log.Println("SystemStatus WriteJSON Failed:", err)
			break
		}

		time.Sleep(1 * time.Second)
	}

	log.Println("Exiting handleWebSocket")

}

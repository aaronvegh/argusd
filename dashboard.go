package main

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

func (app *webSocketApp) handleDashboard(w http.ResponseWriter, r *http.Request) {
	log.Println("This is /connect starting")

	menu := r.URL.Query().Get("menu")
	log.Println("received with isForMenu " + menu)
	isForMenu, err := strconv.ParseBool(menu)

	if err == nil {
		isForMenu = true
	}

	connection, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	session := &webSocketSession{
		connection: connection,
	}

	session.runDashboard(isForMenu)
}

func (ses *webSocketSession) runDashboard(isForMenu bool) {

	for {
		ifaces := interfaces()
		cpuStat := cpuStats()
		memStat := memoryStats()
		diskStat := diskStats()

		payload := dict{
			"memory":     memStat,
			"cpu":        cpuStat,
			"disks":      diskStat,
			"interfaces": ifaces,
		}

		if !isForMenu {
			top5CPUStat := getTop5ProcessesByCPU()
			top5MemStat := getTop5ProcessesByMemory()

			payload["top5cpu"] = top5CPUStat
			payload["top5mem"] = top5MemStat
		}

		if err := ses.connection.WriteJSON(payload); err != nil {
			log.Println("WriteJSON Failed:", err)
			break
		}

		time.Sleep(1 * time.Second)
	}

	log.Println("Exiting handleWebSocket")

}

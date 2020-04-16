package main

import (
	"log"
	"time"
)

func (ses *webSocketSession) runDashboard() {

	for {
		ifaces := interfaces()
		cpuStat := cpuStats()
		memStat := memoryStats()
		diskStat := diskStats()
		top5CPUStat := getTop5ProcessesByCPU()
		top5MemStat := getTop5ProcessesByMemory()

		payload := dict{
			"memory":     memStat,
			"cpu":        cpuStat,
			"disks":      diskStat,
			"top5cpu":    top5CPUStat,
			"top5mem":    top5MemStat,
			"interfaces": ifaces,
		}

		if err := ses.connection.WriteJSON(payload); err != nil {
			log.Println("WriteJSON Failed:", err)
			break
		}

		time.Sleep(1 * time.Second)
	}

	log.Println("Exiting handleWebSocket")

}

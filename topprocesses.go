package main

import (
	"github.com/shirou/gopsutil/process"
	"log"
	"sort"
)

type processStat struct {
	Process string
	Percent float64
}

func getTop5ProcessesByCPU() []processStat {
	processes, err := process.Processes()
	if err != nil {
		log.Println(err)
		return nil
	}

	var finalProcesses = make([]processStat, len(processes))
	for _, ps := range processes {
		percent, err := ps.CPUPercent()
		if err != nil {
			log.Println(err)
			continue
		}

		processName, err := ps.Name()
		if err != nil || processName == "" {
			continue
		}
		finalProcesses = append(finalProcesses, processStat{
			Process: processName,
			Percent: percent,
		})
	}

	sort.Slice(finalProcesses[:], func(i, j int) bool {
		return finalProcesses[i].Percent > finalProcesses[j].Percent
	})

	return finalProcesses[0:5]

}

func getTop5ProcessesByMemory() []processStat {
	processes, err := process.Processes()
	if err != nil {
		log.Println(err)
		return nil
	}

	var finalProcesses = make([]processStat, len(processes))
	for _, ps := range processes {
		memInfo, err := ps.MemoryInfo()
		if err != nil {
			log.Println(err)
			continue
		}

		processName, err := ps.Name()
		if err != nil || processName == "" {
			continue
		}
		finalProcesses = append(finalProcesses, processStat{
			Process: processName,
			Percent: float64(memInfo.RSS),
		})
	}

	sort.Slice(finalProcesses[:], func(i, j int) bool {
		return finalProcesses[i].Percent > finalProcesses[j].Percent
	})

	return finalProcesses[0:5]

}

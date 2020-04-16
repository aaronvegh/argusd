package main

import (
	"github.com/shirou/gopsutil/mem"
	"log"
)

func memoryStats() dict {
	memory, err := mem.VirtualMemory()
	if err != nil {
		log.Println(err)
		return nil
	}
	memStat := dict{
		"MemTotal":         float64(memory.Total),
		"MemFree":          float64(memory.Free),
		"MemUsed":          float64(memory.Used),
		"MemCached":        float64(memory.Cached),
		"MemAvailable":     float64(memory.Available),
		"MemSwapUsed":      float64(memory.SwapCached),
		"MemSwapAvailable": float64(memory.SwapTotal),
	}
	return memStat
}

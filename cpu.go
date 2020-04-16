package main

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"log"
)

var lastStats cpu.TimesStat

func totalCpuTime(t cpu.TimesStat) float64 {
	total := t.User + t.System + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal +
		t.Idle
	return total
}

func cpuStats() dict {
	times, err := cpu.Times(false)
	if err != nil {
		log.Println(err)
		return nil
	}

	var cpuStat dict
	cts := times[0]

	total := totalCpuTime(cts)
	lastCts := lastStats
	lastTotal := totalCpuTime(lastCts)
	totalDelta := total - lastTotal
	if totalDelta < 0 {
		err = fmt.Errorf("Error: current total CPU time is less than previous total CPU time")
		return nil
	}
	if totalDelta == 0 {
		return nil
	}
	usageUser := 100 * (cts.User - lastCts.User - (cts.Guest - lastCts.Guest)) / totalDelta
	usageSystem := 100 * (cts.System - lastCts.System) / totalDelta
	usageIdle := 100 * (cts.Idle - lastCts.Idle) / totalDelta
	cpuStat = dict{
		"User":   usageUser,
		"System": usageSystem,
		"Idle":   usageIdle,
	}

	lastStats = cts

	return cpuStat
}

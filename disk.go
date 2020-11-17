package main

import (
	"log"
	"strings"
	"github.com/shirou/gopsutil/disk"
)

type UsageStat struct {
	Path   string
	Fstype string
	Total  uint64
	Free   uint64
	Used   uint64
}

func diskStats() []UsageStat {
	partitions, err := disk.Partitions(false)
	if err != nil {
		log.Println(err)
		return nil
	}

	var diskUsage []UsageStat
	for _, part := range partitions {
		if !strings.Contains(part.Device, "/loop") {
			disk, err := disk.Usage(part.Mountpoint)
			if err != nil {
				log.Println(err)
				continue
			}
	
			diskUsage = append(diskUsage, UsageStat{
				Path:   disk.Path,
				Fstype: disk.Fstype,
				Total:  disk.Total,
				Free:   disk.Free,
				Used:   disk.Used,
			})
		}
	}
	return diskUsage
}

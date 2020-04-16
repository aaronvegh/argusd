package main

import (
	"github.com/shirou/gopsutil/net"
	"log"
	"regexp"
)

type interfaceStat struct {
	Name      string
	Address   string
	BytesSent uint64
	BytesRecv uint64
}

func interfaces() []interfaceStat {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Println(err)
		return nil
	}

	ioStats, err := net.IOCounters(true)
	if err != nil {
		log.Println(err)
		return nil
	}

	var foundInterfaces []interfaceStat
	for _, iface := range interfaces {
		if len(iface.Addrs) > 0 {
			reg := regexp.MustCompile(`(?:[0-9]{1,3}\.){3}[0-9]{1,3}`)
			var addrString = ""
			for _, addr := range iface.Addrs {
				addrString += addr.Addr
			}

			var foundAddress = reg.FindString(addrString)

			var ifaceCounters = Find(ioStats, iface.Name)

			if len(foundAddress) > 0 && foundAddress != "127.0.0.1" {
				foundInterfaces = append(foundInterfaces, interfaceStat{
					Name:      iface.Name,
					Address:   foundAddress,
					BytesSent: ifaceCounters.BytesSent,
					BytesRecv: ifaceCounters.BytesRecv,
				})
			}
		}
	}

	return foundInterfaces

}

func Find(a []net.IOCountersStat, name string) net.IOCountersStat {
	for _, n := range a {
		if name == n.Name {
			return n
		}
	}
	var counterStat = net.IOCountersStat{}
	return counterStat
}

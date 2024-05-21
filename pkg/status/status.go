// Package status contains all status logic
package status

import (
	"fmt"
	"log"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type Host struct {
}

type Info struct {
	HostName   string `json:"hostname"`
	Procs      int    `json:"procs"`
	HostID     string `json:"host_id"`
	CPUPercent int    `json:"cpu_percent"`
	MemPercent int    `json:"mem_percent"`
	Uptime     uint64 `json:"uptime"`
	Loads      struct {
		One     float64 `json:"one"`
		Five    float64 `json:"five"`
		Fifteen float64 `json:"fifteen"`
	} `json:"load_average"`
}

// Get returns the disk and cpu utilization
func (s Host) Get() (*Info, error) {
	cpup, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get cpu percent: %w", err)
	}

	memp, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory percent: %w", err)
	}

	hostStat, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	loads, err := load.Avg()
	if err != nil {
		return nil, fmt.Errorf("failed to get load average: %w", err)
	}

	res := Info{
		HostName:   hostStat.Hostname,
		Procs:      int(hostStat.Procs),
		HostID:     hostStat.HostID,
		CPUPercent: int(cpup[0]),
		MemPercent: int(memp.UsedPercent),
		Uptime:     hostStat.Uptime,
	}
	res.Loads.One, res.Loads.Five, res.Loads.Fifteen = loads.Load1, loads.Load5, loads.Load15

	log.Printf("[DEBUG] status: %+v", res)
	return &res, nil
}

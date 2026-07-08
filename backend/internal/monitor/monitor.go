package monitor

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type SystemStats struct {
	Timestamp    time.Time      `json:"timestamp"`
	Hostname     string         `json:"hostname"`
	Uptime       uint64         `json:"uptime_seconds"`
	OS           string         `json:"os"`
	Platform     string         `json:"platform"`
	CPU          CPUStats       `json:"cpu"`
	Memory       MemoryStats    `json:"memory"`
	Swap         SwapStats      `json:"swap"`
	Disks        []DiskStats    `json:"disks"`
	Network      NetworkStats   `json:"network"`
	Load         *LoadStats     `json:"load,omitempty"`
	Processes    int32          `json:"process_count"`
	TopProcesses []ProcessInfo  `json:"top_processes,omitempty"`
}

type CPUStats struct {
	PercentUsed float64   `json:"percent_used"`
	PerCore     []float64 `json:"per_core,omitempty"`
	Count       int       `json:"count"`
}

type MemoryStats struct {
	Total     uint64  `json:"total_bytes"`
	Available uint64  `json:"available_bytes"`
	Used      uint64  `json:"used_bytes"`
	Percent   float64 `json:"percent_used"`
}

type SwapStats struct {
	Total   uint64  `json:"total_bytes"`
	Used    uint64  `json:"used_bytes"`
	Percent float64 `json:"percent_used"`
}

type DiskStats struct {
	MountPoint string  `json:"mount_point"`
	FSType     string  `json:"fs_type"`
	Total      uint64  `json:"total_bytes"`
	Used       uint64  `json:"used_bytes"`
	Free       uint64  `json:"free_bytes"`
	Percent    float64 `json:"percent_used"`
}

type NetworkStats struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
}

type LoadStats struct {
	Load1  float64 `json:"load_1"`
	Load5  float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`
}

type ProcessInfo struct {
	PID    int32   `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu_percent"`
	Memory float64 `json:"memory_percent"`
	Status string  `json:"status"`
	Uptime int64   `json:"uptime_seconds"`
}

type procSort struct {
	pid    int32
	name   string
	cpu    float64
	mem    float64
	status string
	uptime int64
}

type Monitor struct {
	mu           sync.RWMutex
	lastNetCount map[string]uint64
	topN         int
}

func New(topN int) *Monitor {
	m := &Monitor{
		topN:         topN,
		lastNetCount: make(map[string]uint64),
	}
	m.captureNetBaseline()
	return m
}

func (m *Monitor) captureNetBaseline() {
	counters, err := net.IOCounters(false)
	if err == nil && len(counters) > 0 {
		m.lastNetCount["bytes_sent"] = counters[0].BytesSent
		m.lastNetCount["bytes_recv"] = counters[0].BytesRecv
	}
}

func (m *Monitor) Gather(ctx context.Context) (*SystemStats, error) {
	stats := &SystemStats{
		Timestamp: time.Now().UTC(),
	}

	// Host info
	if h, err := host.InfoWithContext(ctx); err == nil {
		stats.Hostname = h.Hostname
		stats.Uptime = h.Uptime
		stats.OS = h.OS
		stats.Platform = h.Platform
	} else {
		log.Printf("monitor: host info: %v", err)
	}

	// CPU
	if p, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(p) > 0 {
		stats.CPU.PercentUsed = p[0]
	} else {
		log.Printf("monitor: cpu percent: %v", err)
	}
	stats.CPU.Count = runtime.NumCPU()

	// Memory
	if v, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		stats.Memory = MemoryStats{
			Total:     v.Total,
			Available: v.Available,
			Used:      v.Used,
			Percent:   v.UsedPercent,
		}
	} else {
		log.Printf("monitor: memory: %v", err)
	}

	// Swap
	if s, err := mem.SwapMemoryWithContext(ctx); err == nil {
		stats.Swap = SwapStats{
			Total:   s.Total,
			Used:    s.Used,
			Percent: s.UsedPercent,
		}
	}

	// Disk
	if parts, err := disk.PartitionsWithContext(ctx, false); err == nil {
		for _, p := range parts {
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			stats.Disks = append(stats.Disks, DiskStats{
				MountPoint: p.Mountpoint,
				FSType:     p.Fstype,
				Total:      usage.Total,
				Used:       usage.Used,
				Free:       usage.Free,
				Percent:    usage.UsedPercent,
			})
		}
	} else {
		log.Printf("monitor: disk partitions: %v", err)
	}

	// Network
	if counters, err := net.IOCountersWithContext(ctx, false); err == nil && len(counters) > 0 {
		stats.Network = NetworkStats{
			BytesSent:   counters[0].BytesSent,
			BytesRecv:   counters[0].BytesRecv,
			PacketsSent: counters[0].PacketsSent,
			PacketsRecv: counters[0].PacketsRecv,
		}
	} else {
		log.Printf("monitor: network: %v", err)
	}

	// Load
	if l, err := load.AvgWithContext(ctx); err == nil {
		stats.Load = &LoadStats{
			Load1:  l.Load1,
			Load5:  l.Load5,
			Load15: l.Load15,
		}
	}

	// Process count
	if procs, err := process.ProcessesWithContext(ctx); err == nil {
		stats.Processes = int32(len(procs))
		if m.topN > 0 {
			stats.TopProcesses = m.getTopProcesses(ctx, procs)
		}
	} else {
		log.Printf("monitor: process list: %v", err)
	}

	return stats, nil
}

func (m *Monitor) getTopProcesses(ctx context.Context, procs []*process.Process) []ProcessInfo {
	var results []procSort
	limit := m.topN

	for _, p := range procs {
		if len(results) >= limit*3 {
			break
		}
		name, _ := p.NameWithContext(ctx)
		cpuP, _ := p.CPUPercentWithContext(ctx)
		memP, _ := p.MemoryPercentWithContext(ctx)
		statuses, _ := p.StatusWithContext(ctx)
		createT, _ := p.CreateTimeWithContext(ctx)

		uptime := int64(0)
		if createT > 0 {
			uptime = int64(time.Since(time.Unix(createT/1000, 0)).Seconds())
		}

		statusStr := "unknown"
		if len(statuses) > 0 {
			statusStr = statuses[0]
		}

		results = append(results, procSort{
			pid:    p.Pid,
			name:   name,
			cpu:    cpuP,
			mem:    float64(memP),
			status: statusStr,
			uptime: uptime,
		})
	}

	sortByCPU(results)

	topN := limit
	if len(results) < topN {
		topN = len(results)
	}

	out := make([]ProcessInfo, topN)
	for i := 0; i < topN; i++ {
		out[i] = ProcessInfo{
			PID:    results[i].pid,
			Name:   results[i].name,
			CPU:    results[i].cpu,
			Memory: results[i].mem,
			Status: results[i].status,
			Uptime: results[i].uptime,
		}
	}
	return out
}

func sortByCPU(slice []procSort) {
	for i := 1; i < len(slice); i++ {
		key := slice[i]
		j := i - 1
		for j >= 0 && slice[j].cpu < key.cpu {
			slice[j+1] = slice[j]
			j--
		}
		slice[j+1] = key
	}
}

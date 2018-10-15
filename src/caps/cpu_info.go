package caps

import (
	"fmt"

	"github.com/jaypipes/ghw"
	"github.com/shirou/gopsutil/cpu"
)

// CPUInfo contains the information about an installed CPU
type CPUInfo struct {
	ID             uint32
	Vendor         string
	Product        string
	GHz            float64
	Cores          uint32
	Threads        uint32
	CacheMB        uint32
	HasHardwareAES bool
}

// String converts CPUInfo into a readable string
func (cpu CPUInfo) String() string {
	return fmt.Sprintf(
		"CPU #%d Vendor: %s, Product: %s, Cores: %d, Threads: %d, "+
			"Cache (L3): %d MB, GHz: %.2f, HardwareAES: %t",
		cpu.ID,
		cpu.Vendor,
		cpu.Product,
		cpu.Cores,
		cpu.Threads,
		cpu.CacheMB,
		cpu.GHz,
		cpu.HasHardwareAES,
	)
}

// GetCPUInfo returns information about all installed CPUs
func GetCPUInfo() ([]CPUInfo, error) {
	var cpuInfos []CPUInfo

	cpuDetail, err := ghw.CPU()
	if err != nil {
		return cpuInfos, err
	}
	cpuAdditional, err := cpu.Info()
	if err != nil {
		return cpuInfos, err
	}

	// Get all the info we want for each CPU installed. ghw does not provide
	// cache size, gopsutil does, we loop over all the CPUs to match
	// the IDs and get the additional information
	for _, processor := range cpuDetail.Processors {
		var processorExtended cpu.InfoStat
		for _, processorAdditional := range cpuAdditional {
			if uint32(processorAdditional.CPU) == processor.Id {
				processorExtended = processorAdditional
			}
		}
		cpuInfos = append(cpuInfos, CPUInfo{
			ID:             processor.Id,
			Vendor:         processor.Vendor,
			Product:        processor.Model,
			GHz:            float64(processorExtended.Mhz / 1000.00),
			Cores:          processor.NumCores,
			Threads:        processor.NumThreads,
			CacheMB:        uint32(processorExtended.CacheSize / 1024),
			HasHardwareAES: processor.HasCapability("aes"),
		})
	}

	return cpuInfos, nil
}

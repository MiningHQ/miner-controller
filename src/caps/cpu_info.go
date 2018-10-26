package caps

import (
	"fmt"
	"strconv"
	"strings"

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
	CacheKB        uint32
	HasHardwareAES bool
}

// String converts CPUInfo into a readable string
func (cpu CPUInfo) String() string {
	return fmt.Sprintf(
		"CPU #%d Vendor: %s, Product: %s, Cores: %d, Threads: %d, "+
			"Cache (L3): %d KB, GHz: %.2f, HardwareAES: %t",
		cpu.ID,
		cpu.Vendor,
		cpu.Product,
		cpu.Cores,
		cpu.Threads,
		cpu.CacheKB,
		cpu.GHz,
		cpu.HasHardwareAES,
	)
}

// GetCPUInfo returns information about all installed CPUs
func GetCPUInfo() ([]CPUInfo, error) {
	var cpuInfos []CPUInfo

	cpus, err := cpu.Info()
	if err != nil {
		return cpuInfos, err
	}

	// Get all the info we want for each CPU installed
	// gopsutil provides the CPU information per CPU thread/logical CPU. Thus
	// we need to loop and determine the actual values of each thread's base
	// processorThread.Cores does not return the total value we need
	var currentCPU CPUInfo
	currentCPUID := 0
	for _, processorThread := range cpus {
		cpuID, err := strconv.Atoi(processorThread.PhysicalID)
		if err != nil {
			return cpuInfos, fmt.Errorf("Unable to get CPU ID: %s", err)
		}

		// When we get to a new physical CPU, add the currentCPU
		if currentCPUID != cpuID {
			currentCPU.fixCounts()
			cpuInfos = append(cpuInfos, currentCPU)

			currentCPU = CPUInfo{}
			currentCPUID = cpuID
		}

		currentCPU.ID = uint32(cpuID)
		currentCPU.CacheKB = uint32(processorThread.CacheSize)
		currentCPU.Vendor = processorThread.VendorID
		currentCPU.Product = processorThread.ModelName
		currentCPU.GHz = processorThread.Mhz / 1000
		currentCPU.HasHardwareAES = hasAES(processorThread)
		currentCPU.Threads++

		// coreID is a sequential core list that we can use to get the correct
		// core count for the CPU
		coreID, err := strconv.Atoi(processorThread.CoreID)
		if err != nil {
			return cpuInfos, fmt.Errorf("Unable to get CPU Core ID: %s", err)
		}
		if uint32(coreID) > currentCPU.Cores {
			currentCPU.Cores = uint32(coreID)
		}

	}
	// Add last one as well
	currentCPU.fixCounts()
	cpuInfos = append(cpuInfos, currentCPU)

	return cpuInfos, nil
}

// fixCounts increases cores by one because the IDs start at zero
func (cpu *CPUInfo) fixCounts() {
	cpu.Cores++
}

// hasAES checks if the given CPU Info contains the AES flag
func hasAES(info cpu.InfoStat) bool {
	for _, flag := range info.Flags {
		if strings.Contains(strings.ToLower(flag), "aes") {
			return true
		}
	}
	return false
}

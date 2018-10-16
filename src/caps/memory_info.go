package caps

import (
	"fmt"

	"github.com/shirou/gopsutil/mem"
)

// MemoryInfo contains the information about installed memory
type MemoryInfo struct {
	TotalMB     uint64
	AvailableMB uint64
}

// String converts MemoryInfo into a readable string
func (memory MemoryInfo) String() string {
	return fmt.Sprintf(
		"Memory: Total %d MB, Available %d MB",
		memory.TotalMB,
		memory.AvailableMB)
}

// GetMemoryInfo returns the installed memory in the system
func GetMemoryInfo() (MemoryInfo, error) {
	memory, err := mem.VirtualMemory()
	if err != nil {
		return MemoryInfo{}, err
	}

	// Return the values in megabyte
	return MemoryInfo{
		TotalMB:     memory.Total / 1024 / 1024,
		AvailableMB: memory.Available / 1024 / 1024,
	}, nil
}

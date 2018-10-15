package caps

import (
	"fmt"

	"github.com/jaypipes/ghw"
)

// MemoryInfo contains the information about installed memory
type MemoryInfo struct {
	PhysicalMB int64
	UsableMB   int64
}

// String converts MemoryInfo into a readable string
func (memory MemoryInfo) String() string {
	return fmt.Sprintf(
		"Memory: Physical %d MB, Usable %d MB",
		memory.PhysicalMB,
		memory.UsableMB)
}

// GetMemoryInfo returns the installed memory in the system
func GetMemoryInfo() (MemoryInfo, error) {
	memory, err := ghw.Memory()
	if err != nil {
		return MemoryInfo{}, err
	}
	// Return the values in megabyte
	return MemoryInfo{
		PhysicalMB: memory.TotalPhysicalBytes / 1024 / 1024,
		UsableMB:   memory.TotalUsableBytes / 1024 / 1024,
	}, nil
}

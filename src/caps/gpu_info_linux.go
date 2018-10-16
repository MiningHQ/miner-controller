package caps

import (
	"fmt"

	"github.com/jaypipes/ghw"
)

// GPUInfo contains the information about an installed GPU
type GPUInfo struct {
	ID      uint32
	Vendor  string
	Product string
}

// String converts GPUInfo into a readable string
func (gpu GPUInfo) String() string {
	return fmt.Sprintf(
		"GPU #%d, Vendor: %s, Product: %s",
		gpu.ID,
		gpu.Vendor,
		gpu.Product)
}

// GetGPUInfo returns the installed GPUs in the system
func GetGPUInfo() ([]GPUInfo, error) {
	var gpuInfos []GPUInfo

	gpu, err := ghw.GPU()
	if err != nil {
		return gpuInfos, err
	}

	for _, card := range gpu.GraphicsCards {
		vendor := "Unknown"
		product := "Invalid"
		if card.DeviceInfo != nil {
			if card.DeviceInfo.Vendor != nil {
				vendor = card.DeviceInfo.Vendor.Name
			}
			if card.DeviceInfo.Product != nil {
				product = card.DeviceInfo.Product.Name
			}
		}
		gpuInfos = append(gpuInfos, GPUInfo{
			ID:      uint32(card.Index),
			Vendor:  vendor,
			Product: product,
		})
	}
	return gpuInfos, nil
}

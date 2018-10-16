package caps

import "fmt"

// SystemInfo contains all the collected capabilities of the system
type SystemInfo struct {
	Host   HostInfo
	CPU    []CPUInfo
	Memory MemoryInfo
	GPU    []GPUInfo
}

// String converts SystemInfo into a readable string
func (systemInfo SystemInfo) String() string {
	return fmt.Sprintf(`
System Info
-----------
  Host:
    %s
  Memory:
    %s
  CPUs:
    %s
  GPUs:
    %s
`,
		systemInfo.Host,
		systemInfo.Memory,
		systemInfo.CPU,
		systemInfo.GPU)
}

// GetSystemInfo gathers information about the host, CPUs, Memory and GPUs
func GetSystemInfo() (SystemInfo, error) {
	var systemInfo SystemInfo
	var err error

	systemInfo.Memory, err = GetMemoryInfo()
	if err != nil {
		return systemInfo, err
	}

	systemInfo.CPU, err = GetCPUInfo()
	if err != nil {
		return systemInfo, err
	}

	systemInfo.Host, err = GetHostInfo()
	if err != nil {
		return systemInfo, err
	}

	systemInfo.GPU, err = GetGPUInfo()
	if err != nil {
		return systemInfo, err
	}

	return systemInfo, nil
}

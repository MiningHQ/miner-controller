package caps

import (
	"fmt"

	"github.com/shirou/gopsutil/host"
)

// HostInfo contains the information about the platform and OS
type HostInfo struct {
	ID              string
	Name            string
	OS              string
	Platform        string
	PlatformVersion string
}

// String converts HostInfo into a readable string
func (host HostInfo) String() string {
	return fmt.Sprintf(
		"Host: ID: %s, Name: %s, OS: %s, Platform: %s %s",
		host.ID,
		host.Name,
		host.OS,
		host.Platform,
		host.PlatformVersion)
}

// GetHostInfo returns the information about the OS and platform
func GetHostInfo() (HostInfo, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return HostInfo{}, err
	}
	return HostInfo{
		ID:              hostInfo.HostID,
		Name:            hostInfo.Hostname,
		OS:              hostInfo.OS,
		Platform:        hostInfo.Platform,
		PlatformVersion: hostInfo.PlatformVersion,
	}, nil
}

package protocol

// StaticServerInfo represents static server information sent once at startup
// This structure matches the new API requirements with simplified flat structure
type StaticServerInfo struct {
	Hostname     string       `json:"hostname"`
	OS           string       `json:"os"`
	Kernel       string       `json:"kernel"`
	Architecture string       `json:"architecture"`
	Hardware     HardwareInfo `json:"hardware"`
}

// HardwareInfo represents hardware specifications
type HardwareInfo struct {
	CPUModel            string  `json:"cpu_model"`
	CPUCores            int     `json:"cpu_cores"`
	CPUThreads          int     `json:"cpu_threads"`
	CPUBaseFrequencyMHz float64 `json:"cpu_base_frequency_mhz"`
	MemoryTotalGB       float64 `json:"memory_total_gb"`
}

// Legacy structures kept for backward compatibility
type ServerInfo struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Kernel       string `json:"kernel"`
	Architecture string `json:"architecture"`
}

// StaticNetworkInterface represents static network interface information
type StaticNetworkInterface struct {
	InterfaceName string `json:"interface_name"`
	MACAddress    string `json:"mac_address"`
	InterfaceType string `json:"interface_type"`
	SpeedMbps     int    `json:"speed_mbps,omitempty"`
	IsPhysical    bool   `json:"is_physical"`
}

// DiskStaticInfo represents static disk information
type DiskStaticInfo struct {
	DeviceName     string `json:"device_name"`
	FilesystemType string `json:"filesystem_type"`
	MountPoint     string `json:"mount_point"`
	TotalGB        int    `json:"total_gb"`
}

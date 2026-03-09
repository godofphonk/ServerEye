package protocol

// StaticServerInfo represents static server information sent once per day
type StaticServerInfo struct {
	ServerInfo        ServerInfo               `json:"server_info"`
	HardwareInfo      HardwareInfo             `json:"hardware_info"`
	NetworkInterfaces []StaticNetworkInterface `json:"network_interfaces"`
	DiskInfo          []DiskStaticInfo         `json:"disk_info"`
}

// ServerInfo represents basic server information
type ServerInfo struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Kernel       string `json:"kernel"`
	Architecture string `json:"architecture"`
}

// HardwareInfo represents hardware specifications
type HardwareInfo struct {
	CPUModel        string  `json:"cpu_model"`
	CPUCores        int     `json:"cpu_cores"`
	CPUThreads      int     `json:"cpu_threads"`
	CPUFrequencyMHz float64 `json:"cpu_frequency_mhz"`
	GPUModel        string  `json:"gpu_model,omitempty"`
	GPUDriver       string  `json:"gpu_driver,omitempty"`
	GPUMemoryGB     float64 `json:"gpu_memory_gb,omitempty"`
	TotalMemoryGB   float64 `json:"total_memory_gb"`
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

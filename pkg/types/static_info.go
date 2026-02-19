package types

// StaticInfoRequest represents the request body for static info API
type StaticInfoRequest struct {
	ServerInfo        ServerInfo         `json:"server_info"`
	HardwareInfo      HardwareInfo       `json:"hardware_info"`
	MotherboardInfo   *MotherboardInfo   `json:"motherboard_info,omitempty"`
	MemoryModules     []MemoryModule     `json:"memory_modules,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces,omitempty"`
	DiskInfo          []DiskInfo         `json:"disk_info,omitempty"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Kernel       string `json:"kernel"`
	Architecture string `json:"architecture"`
}

// HardwareInfo represents hardware information
type HardwareInfo struct {
	CPUModel        string  `json:"cpu_model"`
	CPUCores        int     `json:"cpu_cores"`
	CPUThreads      int     `json:"cpu_threads"`
	CPUFrequencyMHz float64 `json:"cpu_frequency_mhz"`
	GPUModel        string  `json:"gpu_model"`
	GPUDriver       string  `json:"gpu_driver"`
	GPUMemoryGB     int     `json:"gpu_memory_gb"`
	TotalMemoryGB   float64 `json:"total_memory_gb"`
}

// NetworkInterface represents network interface information
type NetworkInterface struct {
	InterfaceName string `json:"interface_name"`
	MACAddress    string `json:"mac_address"`
	InterfaceType string `json:"interface_type"`
	SpeedMbps     int    `json:"speed_mbps"`
	Vendor        string `json:"vendor"`
	Driver        string `json:"driver"`
	IsPhysical    bool   `json:"is_physical"`
}

// DiskInfo represents disk information
type DiskInfo struct {
	DeviceName    string `json:"device_name"`
	Model         string `json:"model"`
	SerialNumber  string `json:"serial_number"`
	SizeGB        uint64 `json:"size_gb"`
	DiskType      string `json:"disk_type"`
	InterfaceType string `json:"interface_type"`
	Filesystem    string `json:"filesystem"`
	MountPoint    string `json:"mount_point"`
	IsSystemDisk  bool   `json:"is_system_disk"`
}

// MotherboardInfo represents motherboard information
type MotherboardInfo struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
	Chipset      string `json:"chipset,omitempty"`
}

// MemoryModule represents individual memory module
type MemoryModule struct {
	SlotName     string  `json:"slot_name"`
	SizeGB       uint64  `json:"size_gb"`
	MemoryType   string  `json:"memory_type"`
	FrequencyMHz int     `json:"frequency_mhz"`
	Manufacturer string  `json:"manufacturer"`
	PartNumber   string  `json:"part_number"`
	SpeedMTS     int     `json:"speed_mts"`
	Voltage      float64 `json:"voltage"`
	Timings      string  `json:"timings"`
	ECC          bool    `json:"ecc"`
	Registered   bool    `json:"registered"`
}

// StaticInfoResponse represents the API response
type StaticInfoResponse struct {
	Message  string `json:"message"`
	ServerID string `json:"server_id"`
}

// NetworkInterfaceType represents network interface types
const (
	NetworkTypeEthernet = "ethernet"
	NetworkTypeWiFi     = "wifi"
	NetworkTypeVirtual  = "virtual"
	NetworkTypeLoopback = "loopback"
)

// DiskType represents disk types
const (
	DiskTypeSSD  = "ssd"
	DiskTypeHDD  = "hdd"
	DiskTypeNVMe = "nvme"
	DiskTypeUSB  = "usb"
)

package types

// StaticInfoRequest represents the request body for static info API
type StaticInfoRequest struct {
	ServerInfo        ServerInfo         `json:"server_info"`
	HardwareInfo      HardwareInfo       `json:"hardware_info"`
	MotherboardInfo   MotherboardInfo    `json:"motherboard_info"`
	MemoryInfo        MemoryInfo         `json:"memory_info"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces"`
	DiskInfo          []DiskInfo         `json:"disk_info"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	OSVersion string `json:"os_version"`
}

// HardwareInfo represents hardware information
type HardwareInfo struct {
	CPUModel      string  `json:"cpu_model"`
	CPUCores      int     `json:"cpu_cores"`
	TotalMemoryGB float64 `json:"total_memory_gb"`
}

// NetworkInterface represents network interface information
type NetworkInterface struct {
	InterfaceName string `json:"interface_name"`
	MACAddress    string `json:"mac_address"`
	InterfaceType string `json:"interface_type"`
	SpeedMbps     int    `json:"speed_mbps"`
	IsPhysical    bool   `json:"is_physical"`
}

// DiskInfo represents disk information
type DiskInfo struct {
	DeviceName   string `json:"device_name"`
	Model        string `json:"model"`
	SizeGB       uint64 `json:"size_gb"`
	DiskType     string `json:"disk_type"`
	Filesystem   string `json:"filesystem"`
	MountPoint   string `json:"mount_point"`
	IsSystemDisk bool   `json:"is_system_disk"`
}

// MotherboardInfo represents motherboard information
type MotherboardInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
}

// MemoryInfo represents memory information
type MemoryInfo struct {
	TotalMemoryGB float64        `json:"total_memory_gb"`
	MemoryType    string         `json:"memory_type"`  // DDR3/DDR4/DDR5
	MemorySpeed   int            `json:"memory_speed"` // MHz
	SlotsTotal    int            `json:"slots_total"`
	SlotsUsed     int            `json:"slots_used"`
	Modules       []MemoryModule `json:"modules"`
}

// MemoryModule represents individual memory module
type MemoryModule struct {
	Manufacturer string `json:"manufacturer"`
	SizeGB       uint64 `json:"size_gb"`
	Speed        int    `json:"speed"` // MHz
	Type         string `json:"type"`  // DDR3/DDR4/DDR5
	Slot         int    `json:"slot"`  // Slot number
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

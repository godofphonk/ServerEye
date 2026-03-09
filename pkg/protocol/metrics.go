package protocol

// Metrics represents restructured metrics for WebSocket transmission
type Metrics struct {
	CPUUsage    CPUUsage    `json:"cpu_usage"`
	Memory      Memory      `json:"memory"`
	Disks       []Disk      `json:"disks"`
	Network     Network     `json:"network"`
	Temperature Temperature `json:"temperature"`
	System      System      `json:"system"`
	Timestamp   string      `json:"timestamp"`
}

// CPUUsage represents CPU usage metrics
type CPUUsage struct {
	UsageTotal   float64     `json:"usage_total"`
	UsageUser    float64     `json:"usage_user"`
	UsageSystem  float64     `json:"usage_system"`
	UsageIdle    float64     `json:"usage_idle"`
	LoadAverage  LoadAverage `json:"load_average"`
	FrequencyMHz float64     `json:"frequency_mhz"`
}

// LoadAverage represents system load averages
type LoadAverage struct {
	Load1Min  float64 `json:"load_1min"`
	Load5Min  float64 `json:"load_5min"`
	Load15Min float64 `json:"load_15min"`
}

// Memory represents memory metrics
type Memory struct {
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	AvailableGB float64 `json:"available_gb"`
	FreeGB      float64 `json:"free_gb"`
	BuffersGB   float64 `json:"buffers_gb"`
	CachedGB    float64 `json:"cached_gb"`
	UsedPercent float64 `json:"used_percent"`
}

// Disk represents disk metrics
type Disk struct {
	MountPoint  string  `json:"mount_point"`
	DeviceName  string  `json:"device_name"`
	UsedGB      int     `json:"used_gb"`
	FreeGB      int     `json:"free_gb"`
	UsedPercent float64 `json:"used_percent"`
}

// Network represents network metrics
type Network struct {
	Interfaces  []NetworkInterface `json:"interfaces"`
	TotalRxMbps float64            `json:"total_rx_mbps"`
	TotalTxMbps float64            `json:"total_tx_mbps"`
}

// NetworkInterface represents network interface metrics
type NetworkInterface struct {
	Name        string  `json:"name"`
	RxBytes     int64   `json:"rx_bytes"`
	TxBytes     int64   `json:"tx_bytes"`
	RxSpeedMbps float64 `json:"rx_speed_mbps"`
	TxSpeedMbps float64 `json:"tx_speed_mbps"`
	Status      string  `json:"status"`
}

// Temperature represents temperature metrics
type Temperature struct {
	CPU     float64       `json:"cpu"`
	GPU     float64       `json:"gpu"`
	Storage []StorageTemp `json:"storage"`
	Highest float64       `json:"highest"`
}

// StorageTemp represents storage device temperature
type StorageTemp struct {
	Device      string  `json:"device"`
	Temperature float64 `json:"temperature"`
}

// System represents system information
type System struct {
	ProcessesTotal    int   `json:"processes_total"`
	ProcessesRunning  int   `json:"processes_running"`
	ProcessesSleeping int   `json:"processes_sleeping"`
	UptimeSeconds     int64 `json:"uptime_seconds"`
}

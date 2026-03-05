package metrics

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// GPUInfo represents GPU information
type GPUInfo struct {
	Model    string
	Driver   string
	MemoryMB int
	MemoryGB float64
}

// GPUMetrics handles GPU metrics collection
type GPUMetrics struct{}

// NewGPUMetrics creates a new GPU metrics collector
func NewGPUMetrics() *GPUMetrics {
	return &GPUMetrics{}
}

// GetGPUInfo retrieves GPU information from the system
func (g *GPUMetrics) GetGPUInfo() (*GPUInfo, error) {
	// Try nvidia-smi first for NVIDIA GPUs (best for memory info)
	if info, err := g.getGPUInfoFromNvidia(); err == nil && info.Model != "" {
		return info, nil
	}

	// Try lspci for general GPU detection
	if info, err := g.getGPUInfoFromLspci(); err == nil && info.Model != "" {
		// If we got GPU from lspci but it's NVIDIA, try to get memory from nvidia-smi
		if strings.Contains(strings.ToLower(info.Model), "nvidia") {
			if nvidiaInfo, err := g.getGPUInfoFromNvidia(); err == nil {
				info.Driver = nvidiaInfo.Driver
				info.MemoryMB = nvidiaInfo.MemoryMB
				info.MemoryGB = nvidiaInfo.MemoryGB
			}
		}
		return info, nil
	}

	if info, err := g.getGPUInfoFromAMD(); err == nil && info.Model != "" {
		return info, nil
	}

	if info, err := g.getGPUInfoFromIntel(); err == nil && info.Model != "" {
		return info, nil
	}

	// Return empty GPU info if no GPU found
	return &GPUInfo{
		Model:    "",
		Driver:   "",
		MemoryMB: 0,
		MemoryGB: 0,
	}, nil
}

// getGPUInfoFromLspci tries to get GPU info from lspci command
func (g *GPUMetrics) getGPUInfoFromLspci() (*GPUInfo, error) {
	cmd := exec.Command("lspci", "-v")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lspci: %w", err)
	}

	info := &GPUInfo{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	inGPUSection := false

	for scanner.Scan() {
		line := scanner.Text()

		// Look for VGA/3D controller lines
		if strings.Contains(line, "VGA compatible controller") ||
			strings.Contains(line, "3D controller") ||
			strings.Contains(line, "Display controller") {
			inGPUSection = true
			// Extract model name
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				model := strings.TrimSpace(parts[1])
				// Remove kernel driver info if present
				if idx := strings.Index(model, "["); idx != -1 {
					model = strings.TrimSpace(model[:idx])
				}
				// Clean up model name - remove controller prefixes
				model = strings.ReplaceAll(model, "VGA compatible controller: ", "")
				model = strings.ReplaceAll(model, "3D controller: ", "")
				model = strings.ReplaceAll(model, "Display controller: ", "")
				model = strings.TrimSpace(model)
				info.Model = model
			}
			continue
		}

		if inGPUSection && strings.HasPrefix(line, "\t") {
			// Extract driver
			if strings.Contains(line, "Kernel driver in use:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.Driver = strings.TrimSpace(parts[1])
				}
			}

			// Extract memory size (for some GPUs)
			if strings.Contains(line, "Memory at") && strings.Contains(line, "size=") {
				re := regexp.MustCompile(`size=(\d+)`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					if sizeKB, err := strconv.Atoi(matches[1]); err == nil {
						info.MemoryMB = sizeKB / 1024
						info.MemoryGB = float64(info.MemoryMB) / 1024
					}
				}
			}
		}

		// Reset when we hit a new device section
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") && line != "" {
			if inGPUSection && info.Model != "" {
				break // We got our GPU info
			}
			inGPUSection = false
		}
	}

	if info.Model == "" {
		return nil, fmt.Errorf("no GPU found in lspci output")
	}

	return info, nil
}

// getGPUInfoFromNvidia tries to get GPU info from nvidia-smi
func (g *GPUMetrics) getGPUInfoFromNvidia() (*GPUInfo, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,driver_version,memory.total", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run nvidia-smi: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid nvidia-smi output format")
	}

	info := &GPUInfo{
		Model:  strings.TrimSpace(parts[0]),
		Driver: strings.TrimSpace(parts[1]),
	}

	if memoryMB, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
		info.MemoryMB = memoryMB
		// Round to nearest GB
		info.MemoryGB = float64(memoryMB) / 1024
		roundedGB := int(info.MemoryGB + 0.5)
		info.MemoryGB = float64(roundedGB)
	}

	return info, nil
}

// getGPUInfoFromAMD tries to get GPU info from AMD tools
func (g *GPUMetrics) getGPUInfoFromAMD() (*GPUInfo, error) {
	// Try rocm-smi for AMD GPUs
	cmd := exec.Command("rocm-smi", "--showproductname")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run rocm-smi: %w", err)
	}

	info := &GPUInfo{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Card series") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Model = strings.TrimSpace(parts[1])
				info.Driver = "amdgpu"
			}
		}
	}

	if info.Model == "" {
		return nil, fmt.Errorf("no AMD GPU found")
	}

	return info, nil
}

// getGPUInfoFromIntel tries to get GPU info from Intel GPUs
func (g *GPUMetrics) getGPUInfoFromIntel() (*GPUInfo, error) {
	// Intel integrated GPUs are usually detected by lspci
	// This is a fallback for Intel-specific detection
	cmd := exec.Command("lspci")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lspci: %w", err)
	}

	info := &GPUInfo{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Intel") &&
			(strings.Contains(line, "VGA") || strings.Contains(line, "Display")) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				model := strings.TrimSpace(parts[1])
				info.Model = model
				info.Driver = "i915" // Common Intel GPU driver
			}
			break
		}
	}

	if info.Model == "" {
		return nil, fmt.Errorf("no Intel GPU found")
	}

	return info, nil
}

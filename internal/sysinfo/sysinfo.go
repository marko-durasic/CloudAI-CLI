package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemSpecs represents the detected system specifications
type SystemSpecs struct {
	CPUCores int
	RAMGB    int
	HasGPU   bool
	GPUType  string
}

// DetectSystemSpecs detects the current system specifications
func DetectSystemSpecs() (*SystemSpecs, error) {
	specs := &SystemSpecs{
		CPUCores: runtime.NumCPU(),
	}

	// Detect RAM
	ramGB, err := detectRAM()
	if err != nil {
		return nil, fmt.Errorf("failed to detect RAM: %w", err)
	}
	specs.RAMGB = ramGB

	// Detect GPU
	hasGPU, gpuType, err := detectGPU()
	if err != nil {
		// Don't fail on GPU detection, just log it
		fmt.Fprintf(os.Stderr, "Warning: GPU detection failed: %v\n", err)
	}
	specs.HasGPU = hasGPU
	specs.GPUType = gpuType

	return specs, nil
}

// detectRAM detects available RAM in GB
func detectRAM() (int, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("could not open /proc/meminfo: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// MemTotal is in KB, convert to GB
				memKB, err := strconv.Atoi(parts[1])
				if err != nil {
					return 0, fmt.Errorf("could not parse memory value: %w", err)
				}
				return memKB / 1024 / 1024, nil // Convert KB to GB
			}
		}
	}

	return 0, fmt.Errorf("could not find MemTotal in /proc/meminfo")
}

// detectGPU detects if a GPU is available and its type
func detectGPU() (bool, string, error) {
	// Check for NVIDIA GPU
	file, err := os.Open("/proc/driver/nvidia/version")
	if err == nil {
		defer file.Close()
		return true, "NVIDIA", nil
	}

	// Check via lspci for any GPU
	// For now, we'll just check if nvidia-smi exists
	_, err = os.Stat("/usr/bin/nvidia-smi")
	if err == nil {
		return true, "NVIDIA", nil
	}

	// Check for Intel/AMD GPUs via lspci
	// This is a simplified check - in a real implementation, you'd parse lspci output
	return false, "", nil
}

// String returns a human-readable representation of system specs
func (s *SystemSpecs) String() string {
	gpuInfo := "No GPU"
	if s.HasGPU {
		gpuInfo = fmt.Sprintf("%s GPU", s.GPUType)
	}
	return fmt.Sprintf("CPU: %d cores, RAM: %d GB, %s", s.CPUCores, s.RAMGB, gpuInfo)
}

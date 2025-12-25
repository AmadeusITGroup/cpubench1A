//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// NumaNodeInfo represents NUMA node information (cross-platform)
type NumaNodeInfo struct {
	NodeNumber uint32
	CPUCount   int
}

// GetNumaTopology retrieves NUMA topology information for Unix-like systems
func GetNumaTopology() ([]NumaNodeInfo, error) {
	// This uses the existing /sys filesystem approach
	return getNumaTopologyFromSys()
}

// getNumaTopologyFromSys reads NUMA topology from /sys filesystem (Linux)
func getNumaTopologyFromSys() ([]NumaNodeInfo, error) {
	numaPath := "/sys/devices/system/node"
	
	entries, err := os.ReadDir(numaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read NUMA directory: %v", err)
	}

	var nodes []NumaNodeInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "node") {
			continue
		}

		// Extract node number
		nodeNumStr := strings.TrimPrefix(name, "node")
		nodeNum, err := strconv.ParseUint(nodeNumStr, 10, 32)
		if err != nil {
			continue
		}

		// Count CPUs in this node
		cpuListPath := filepath.Join(numaPath, name, "cpulist")
		cpuCount := 0
		
		if data, err := os.ReadFile(cpuListPath); err == nil {
			cpuList := strings.TrimSpace(string(data))
			cpuCount = countCPUsInList(cpuList)
		}

		nodes = append(nodes, NumaNodeInfo{
			NodeNumber: uint32(nodeNum),
			CPUCount:   cpuCount,
		})
	}

	return nodes, nil
}

// countCPUsInList counts CPUs from a CPU list string (e.g., "0-3,8-11")
func countCPUsInList(cpuList string) int {
	if cpuList == "" {
		return 0
	}

	count := 0
	ranges := strings.Split(cpuList, ",")
	
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}

		if strings.Contains(r, "-") {
			parts := strings.Split(r, "-")
			if len(parts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil {
					count += (end - start + 1)
				}
			}
		} else {
			count++
		}
	}

	return count
}

// GetNumaTopologyString returns a formatted string describing the NUMA topology
func GetNumaTopologyString() string {
	nodes, err := GetNumaTopology()
	if err != nil {
		return fmt.Sprintf("NUMA topology unavailable: %v", err)
	}

	if len(nodes) == 0 {
		return "No NUMA nodes detected (UMA system)"
	}

	result := fmt.Sprintf("NUMA Nodes: %d\n", len(nodes))
	for _, node := range nodes {
		result += fmt.Sprintf("  Node %d: %d CPUs\n", node.NodeNumber, node.CPUCount)
	}

	return result
}

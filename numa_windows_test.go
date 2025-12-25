//go:build windows

package main

import (
	"testing"
)

func TestGetNumaHighestNodeNumber(t *testing.T) {
	nodeNum, err := GetNumaHighestNodeNumber()
	if err != nil {
		t.Logf("GetNumaHighestNodeNumber failed (may be expected on non-NUMA systems): %v", err)
		return
	}
	t.Logf("Highest NUMA node number: %d", nodeNum)
}

func TestGetNumaTopology(t *testing.T) {
	nodes, err := GetNumaTopology()
	if err != nil {
		t.Fatalf("GetNumaTopology failed: %v", err)
	}

	if len(nodes) == 0 {
		t.Log("No NUMA nodes detected (UMA system)")
		return
	}

	t.Logf("Found %d NUMA nodes:", len(nodes))
	for _, node := range nodes {
		t.Logf("  Node %d: %d CPUs (Group %d, Mask 0x%x)",
			node.NodeNumber, node.CPUCount, node.GroupMask.Group, node.GroupMask.Mask)
	}
}

func TestCountBits(t *testing.T) {
	tests := []struct {
		mask     uint64
		expected int
	}{
		{0x0, 0},
		{0x1, 1},
		{0x3, 2},
		{0xF, 4},
		{0xFF, 8},
		{0xFFFF, 16},
		{0xFFFFFFFF, 32},
		{0xFFFFFFFFFFFFFFFF, 64},
		{0x5, 2},  // 0101
		{0xA, 2},  // 1010
		{0x55, 4}, // 01010101
	}

	for _, tt := range tests {
		result := countBits(tt.mask)
		if result != tt.expected {
			t.Errorf("countBits(0x%x) = %d, expected %d", tt.mask, result, tt.expected)
		}
	}
}

//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetNumaHighestNodeNumber         = kernel32.NewProc("GetNumaHighestNodeNumber")
	procGetLogicalProcessorInformationEx = kernel32.NewProc("GetLogicalProcessorInformationEx")
	procGetNumaNodeProcessorMaskEx       = kernel32.NewProc("GetNumaNodeProcessorMaskEx")
	procGetNumaProcessorNodeEx           = kernel32.NewProc("GetNumaProcessorNodeEx")
)

// Windows API constants
const (
	RelationProcessorCore    = 0
	RelationNumaNode         = 1
	RelationCache            = 2
	RelationProcessorPackage = 3
	RelationGroup            = 4
	RelationAll              = 0xffff
)

// GROUP_AFFINITY structure
type GROUP_AFFINITY struct {
	Mask     uint64
	Group    uint16
	Reserved [3]uint16
}

// NUMA_NODE_RELATIONSHIP structure
type NUMA_NODE_RELATIONSHIP struct {
	NodeNumber uint32
	Reserved   [18]byte
	GroupCount uint16
	GroupMask  GROUP_AFFINITY
	// Note: In reality, GroupMasks is a variable-length array, but we only use GroupMask for simplicity
}

// PROCESSOR_RELATIONSHIP structure
type PROCESSOR_RELATIONSHIP struct {
	Flags           byte
	EfficiencyClass byte
	Reserved        [20]byte
	GroupCount      uint16
	GroupMask       [1]GROUP_AFFINITY // Variable length array
}

// CACHE_RELATIONSHIP structure
type CACHE_RELATIONSHIP struct {
	Level         byte
	Associativity byte
	LineSize      uint16
	CacheSize     uint32
	Type          int32
	Reserved      [18]byte
	GroupCount    uint16
	GroupMask     GROUP_AFFINITY
}

// GROUP_RELATIONSHIP structure
type GROUP_RELATIONSHIP struct {
	MaximumGroupCount      uint16
	ActiveGroupCount       uint16
	Reserved               [20]byte
	GroupInfo              [1]PROCESSOR_GROUP_INFO // Variable length array
}

// PROCESSOR_GROUP_INFO structure
type PROCESSOR_GROUP_INFO struct {
	MaximumProcessorCount byte
	ActiveProcessorCount  byte
	Reserved              [38]byte
	ActiveProcessorMask   uint64
}

// SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX structure
type SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX struct {
	Relationship uint32
	Size         uint32
	// Union of different relationship types
	// We'll handle this with unsafe pointer arithmetic
}

// GetNumaHighestNodeNumber retrieves the highest NUMA node number
func GetNumaHighestNodeNumber() (uint32, error) {
	var highestNodeNumber uint32
	ret, _, err := procGetNumaHighestNodeNumber.Call(
		uintptr(unsafe.Pointer(&highestNodeNumber)),
	)
	if ret == 0 {
		return 0, fmt.Errorf("GetNumaHighestNodeNumber failed: %v", err)
	}
	return highestNodeNumber, nil
}

// GetLogicalProcessorInformationEx retrieves processor topology information
func GetLogicalProcessorInformationEx(relationshipType uint32) ([]byte, error) {
	var bufferSize uint32 = 0

	// First call to get required buffer size
	ret, _, _ := procGetLogicalProcessorInformationEx.Call(
		uintptr(relationshipType),
		0, // NULL buffer
		uintptr(unsafe.Pointer(&bufferSize)),
	)

	if ret != 0 {
		return nil, fmt.Errorf("unexpected success on first call")
	}

	// Allocate buffer
	buffer := make([]byte, bufferSize)

	// Second call to get actual data
	ret, _, err := procGetLogicalProcessorInformationEx.Call(
		uintptr(relationshipType),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&bufferSize)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("GetLogicalProcessorInformationEx failed: %v", err)
	}

	return buffer[:bufferSize], nil
}

// NumaNodeInfo represents NUMA node information
type NumaNodeInfo struct {
	NodeNumber uint32
	GroupMask  GROUP_AFFINITY
	CPUCount   int
}

// GetNumaTopology retrieves NUMA topology information for Windows
func GetNumaTopology() ([]NumaNodeInfo, error) {
	// Check if system supports NUMA
	highestNode, err := GetNumaHighestNodeNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to get highest NUMA node: %v", err)
	}

	// If highestNode is 0, it could be a single-node system or non-NUMA
	// We'll still try to get the information

	// Get NUMA node information
	buffer, err := GetLogicalProcessorInformationEx(RelationNumaNode)
	if err != nil {
		return nil, fmt.Errorf("failed to get NUMA information: %v", err)
	}

	var nodes []NumaNodeInfo
	offset := 0

	for offset < len(buffer) {
		if offset+8 > len(buffer) {
			break
		}

		// Read the header
		info := (*SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX)(unsafe.Pointer(&buffer[offset]))

		if info.Relationship == RelationNumaNode {
			// The NUMA_NODE_RELATIONSHIP starts after the header (8 bytes)
			if offset+int(info.Size) <= len(buffer) {
				numaNode := (*NUMA_NODE_RELATIONSHIP)(unsafe.Pointer(&buffer[offset+8]))
				
				// Count CPUs in the group mask
				cpuCount := countBits(numaNode.GroupMask.Mask)

				nodes = append(nodes, NumaNodeInfo{
					NodeNumber: numaNode.NodeNumber,
					GroupMask:  numaNode.GroupMask,
					CPUCount:   cpuCount,
				})
			}
		}

		offset += int(info.Size)
	}

	// Validate we got expected number of nodes
	if len(nodes) > 0 && uint32(len(nodes)-1) != highestNode {
		// This is informational, not an error
		fmt.Printf("Warning: Expected %d nodes based on highest node number, but found %d nodes\n", 
			highestNode+1, len(nodes))
	}

	return nodes, nil
}

// countBits counts the number of set bits in a uint64
func countBits(mask uint64) int {
	count := 0
	for mask != 0 {
		count++
		mask &= mask - 1
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
		result += fmt.Sprintf("  Node %d: %d CPUs (Group %d, Mask 0x%x)\n",
			node.NodeNumber, node.CPUCount, node.GroupMask.Group, node.GroupMask.Mask)
	}

	return result
}

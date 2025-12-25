//go:build !windows

package main

import (
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
)

// DisplayNumaTopology displays the NUMA CPU topology
func DisplayNumaTopology(cpuinfo []cpu.InfoStat) {

	// NUMA topology retrieval only works on Linux
	if files, err := filepath.Glob("/sys/devices/system/node/node[0-9]*/cpu[0-9]*"); err == nil && len(files) > 0 {

		// Fetch NUMA topology
		numa := map[int]int{}
		for _, f := range files {
			t := strings.Split(strings.TrimPrefix(f, "/sys/devices/system/node/"), "/")
			if len(t) > 1 {
				var n, c int
				if n, err = strconv.Atoi(strings.TrimPrefix(t[0], "node")); err != nil {
					continue
				}
				if c, err = strconv.Atoi(strings.TrimPrefix(t[1], "cpu")); err != nil {
					continue
				}
				numa[c] = n
			}
		}

		// Display NUMA topology
		for _, c := range cpuinfo {
			n, ok := numa[int(c.CPU)]
			if !ok {
				n = -1
			}
			log.Printf("CPU:%3d Socket:%3s CoreId:%3s Node:%3d", c.CPU, c.PhysicalID, c.CoreID, n)
		}
	}
}

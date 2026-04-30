package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run check_memory.go <path-to-mem.out>")
		os.Exit(1)
	}

	memProfilePath := os.Args[1]

	// Run go tool pprof to get the top allocations in bytes
	cmd := exec.Command("go", "tool", "pprof", "-alloc_space", "-top", "-unit=B", memProfilePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running pprof: %s\n%s\n", err, out)
		os.Exit(1)
	}

	// Parse the output by scanning for the summary line
	lines := strings.Split(string(out), "\n")
	var summaryLine string
	for _, line := range lines {
		if strings.Contains(line, "Showing nodes accounting for") {
			summaryLine = line
			break
		}
	}

	if summaryLine == "" {
		fmt.Printf("Could not find the memory summary line in pprof output.\nOutput was:\n%s\n", out)
		os.Exit(1)
	}

	parts := strings.Split(summaryLine, " of ")
	if len(parts) < 2 {
		fmt.Printf("Could not parse total memory from summary line: %s\n", summaryLine)
		os.Exit(1)
	}

	totalStr := strings.TrimSpace(parts[1])
	totalStr = strings.TrimSuffix(totalStr, " total")
	totalStr = strings.TrimSuffix(totalStr, "B") // remove the 'B' unit

	totalBytes, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil {
		fmt.Printf("Could not convert total memory to integer: %s\n", err)
		os.Exit(1)
	}

	totalMB := totalBytes / (1024 * 1024)
	fmt.Printf("Peak Memory Allocation Detected: %d MB\n", totalMB)

	if totalMB > 2048 {
		fmt.Printf("FAILURE: Memory usage (%d MB) spiked above the 2048 MB (2GB) limit.\n", totalMB)
		os.Exit(1)
	}

	fmt.Println("✅ PASS: Memory usage is within the 2GB limit.")
}
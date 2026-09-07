// faultprobe is copied only into the integration image, never the production processor.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var retained [][]byte

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("qpdf version 12.4.1")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--suppress-recovery" {
		_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
		os.Exit(2)
	}

	if len(os.Args) == 2 && os.Args[1] == "events" {
		data, err := os.ReadFile("/sys/fs/cgroup/memory.events")
		if err != nil {
			os.Exit(2)
		}
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "oom_kill" {
				value, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					os.Exit(2)
				}
				fmt.Println(value)
				return
			}
		}
		os.Exit(2)
	}
	if len(os.Args) != 2 || os.Args[1] != "exhaust" {
		os.Exit(2)
	}
	for {
		block := make([]byte, 1024*1024)
		for i := 0; i < len(block); i += 4096 {
			block[i] = 1
		}
		retained = append(retained, block)
	}
}

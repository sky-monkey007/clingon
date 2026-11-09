//go:build linux

package tmux

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

// foregroundGroup reads /proc/[pid]/stat to get the terminal process group ID (tpgid, field 8).
func foregroundGroup(pid int) (int, error) {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, err
	}
	fields := bytes.Fields(stat)
	if len(fields) >= 8 {
		tpgid, _ := strconv.Atoi(string(fields[7]))
		return tpgid, nil
	}
	return 0, fmt.Errorf("could not parse tpgid")
}

// groupMember iterates through /proc to find a non-zombie process matching the pgid.
func groupMember(pgid int) (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || !entry.IsDir() {
			continue
		}
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}
		fields := bytes.Fields(stat)
		if len(fields) >= 5 {
			pgrp, _ := strconv.Atoi(string(fields[4]))
			state := string(fields[2])
			if pgrp == pgid && state != "Z" { // Not a zombie
				return pid, nil
			}
		}
	}
	return 0, unix.ESRCH
}

// flushInput discards keys typed on the terminal but not yet read.
func flushInput(fd int) error {
	return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIFLUSH)
}

// waitExit polls to check if the process is still running.
func waitExit(pid int) error {
	for {
		err := unix.Kill(pid, 0)
		if err != nil { // ESRCH means process is gone
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

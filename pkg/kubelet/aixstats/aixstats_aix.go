//go:build aix

/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aixstats

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	cadvisorapi "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

// getsystemcfg labels, from /usr/include/sys/systemcfg.h.
const (
	scNCPUS   = 5  // number of CPUs active on the system (logical, including SMT threads)
	scPhysMem = 33 // bytes of OS available memory
)

const commandTimeout = 10 * time.Second

// BootTime returns the time the system booted, to the minute, from the boot
// record in /etc/utmp. The result is cached for the life of the process, so
// every caller sees the same value.
func BootTime() (time.Time, error) {
	return bootTime()
}

var bootTime = sync.OnceValues(func() (time.Time, error) {
	out, err := run("/usr/bin/who", []string{"TZ=UTC0"}, "-b")
	if err != nil {
		return time.Time{}, err
	}
	return parseWhoBoot(out, time.Now())
})

// MachineInfo returns the node's CPU count, memory and identifiers.
//
// CPU and memory come from getsystemcfg and must succeed. The identifiers
// are best effort: a failure is logged and the field falls back or is left
// empty, because they are informational in the node status.
func MachineInfo(logger klog.Logger) (*cadvisorapi.MachineInfo, error) {
	numCores := int(unix.Getsystemcfg(scNCPUS))
	if numCores <= 0 {
		numCores = runtime.NumCPU()
	}
	memory := unix.Getsystemcfg(scPhysMem)
	if memory == 0 {
		return nil, fmt.Errorf("getsystemcfg(SC_PHYSMEM) returned 0")
	}

	// os_uuid identifies the AIX installation, which is what Linux's
	// /etc/machine-id identifies. It is also the best available stand-in for
	// the hardware UUID that Linux reads from DMI.
	osUUID := ""
	if out, err := run("/usr/sbin/lsattr", nil, "-E", "-l", "sys0", "-a", "os_uuid", "-F", "value"); err != nil {
		logger.Info("Could not read the AIX os_uuid; using the hostname as the machine ID", "err", err)
	} else {
		osUUID = parseLsattrValue(out)
	}
	machineID := osUUID
	if machineID == "" {
		if hostname, err := os.Hostname(); err == nil {
			machineID = hostname
		}
	}

	bootID := ""
	if bt, err := BootTime(); err != nil {
		logger.Info("Could not read the AIX boot time; the node will report no boot ID", "err", err)
	} else {
		bootID = bootIDFromTime(bt)
	}

	return &cadvisorapi.MachineInfo{
		Timestamp:      time.Now(),
		NumCores:       numCores,
		MemoryCapacity: memory,
		MachineID:      machineID,
		SystemUUID:     osUUID,
		BootID:         bootID,
	}, nil
}

// VersionInfo returns the AIX kernel version and OS image. oslevel is best
// effort; uname must succeed.
func VersionInfo(logger klog.Logger) (*cadvisorapi.VersionInfo, error) {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return nil, fmt.Errorf("uname: %w", err)
	}
	kv := kernelVersion(cstring(u.Version[:]), cstring(u.Release[:]))

	osLevel, err := run("/usr/bin/oslevel", nil, "-s")
	if err != nil {
		logger.Info("Could not read the AIX service pack level", "err", err)
		osLevel = ""
	}

	return &cadvisorapi.VersionInfo{
		KernelVersion:      kv,
		ContainerOsVersion: osImage(kv, osLevel),
	}, nil
}

// DirFsInfo returns capacity, usage and inode counts for the file system
// that holds path.
func DirFsInfo(path string) (cadvisorapiv2.FsInfo, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return cadvisorapiv2.FsInfo{}, fmt.Errorf("statfs %q: %w", path, err)
	}
	// f_blocks, f_bfree and f_bavail are in units of f_bsize, as df uses them.
	capacity := st.Blocks * st.Bsize
	available := st.Bavail * st.Bsize
	usage := (st.Blocks - st.Bfree) * st.Bsize
	inodes := st.Files
	inodesFree := st.Ffree
	return cadvisorapiv2.FsInfo{
		Timestamp:  time.Now(),
		Mountpoint: path,
		Capacity:   capacity,
		Available:  available,
		Usage:      usage,
		Inodes:     &inodes,
		InodesFree: &inodesFree,
	}, nil
}

// run runs an AIX command with a C locale and returns its standard output.
func run(name string, extraEnv []string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(append(os.Environ(), "LC_ALL=C"), extraEnv...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}

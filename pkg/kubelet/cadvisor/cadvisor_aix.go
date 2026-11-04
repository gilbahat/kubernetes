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

package cadvisor

import (
	"context"
	"fmt"
	"time"

	cadvisorapi "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/aixstats"
)

// cadvisorClient implements Interface on AIX, which has no cgroups and no
// cAdvisor. It reports the machine's capacity and identity and file system
// usage. Container and pod stats come from the CRI runtime instead.
type cadvisorClient struct {
	rootPath    string
	machineInfo *cadvisorapi.MachineInfo
	versionInfo *cadvisorapi.VersionInfo
	bootTime    time.Time
}

var _ Interface = new(cadvisorClient)

// New reads the machine and version info once; neither changes while the
// kubelet runs, except through dynamic LPAR changes, which need a kubelet
// restart to be picked up, as on Linux.
func New(logger klog.Logger, imageFsInfoProvider ImageFsInfoProvider, rootPath string, cgroupRoots []string, usingLegacyStats, localStorageCapacityIsolation bool) (Interface, error) {
	machineInfo, err := aixstats.MachineInfo(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to read AIX machine info: %w", err)
	}
	versionInfo, err := aixstats.VersionInfo(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to read AIX version info: %w", err)
	}
	// Boot time is best effort; aixstats.MachineInfo already logged a failure.
	bootTime, _ := aixstats.BootTime()
	return &cadvisorClient{
		rootPath:    rootPath,
		machineInfo: machineInfo,
		versionInfo: versionInfo,
		bootTime:    bootTime,
	}, nil
}

func (cu *cadvisorClient) Start() error {
	return nil
}

// ContainerInfoV2 only knows the root container "/", which stands for the
// node. It reports no CPU or memory usage yet, so the summary API returns a
// node entry without those stats instead of failing. Other names are cgroup
// paths, which do not exist on AIX.
//
// TODO(aix): report node CPU and memory usage (libperfstat or vmstat).
func (cu *cadvisorClient) ContainerInfoV2(name string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerInfo, error) {
	if name != "/" {
		return nil, fmt.Errorf("container %q: AIX has no cgroups; only the root container is available", name)
	}
	return map[string]cadvisorapiv2.ContainerInfo{
		"/": {
			Spec: cadvisorapiv2.ContainerSpec{
				CreationTime: cu.bootTime,
				HasCpu:       false,
				HasMemory:    false,
				HasNetwork:   false,
				Memory: cadvisorapiv2.MemorySpec{
					Limit: cu.machineInfo.MemoryCapacity,
				},
			},
			Stats: []*cadvisorapiv2.ContainerStats{{Timestamp: time.Now()}},
		},
	}, nil
}

func (cu *cadvisorClient) GetRequestedContainersInfo(containerName string, options cadvisorapiv2.RequestOptions) (map[string]*cadvisorapi.ContainerInfo, error) {
	return nil, nil
}

// MachineInfo returns a copy of the cached machine info, so callers cannot
// change the cache.
func (cu *cadvisorClient) MachineInfo() (*cadvisorapi.MachineInfo, error) {
	mi := *cu.machineInfo
	return &mi, nil
}

func (cu *cadvisorClient) VersionInfo() (*cadvisorapi.VersionInfo, error) {
	vi := *cu.versionInfo
	return &vi, nil
}

// ImagesFsInfo and ContainerFsInfo are empty, as on Windows: the CRI stats
// provider asks the runtime for image file system usage and then calls
// GetDirFsInfo on the mount point the runtime reports.
func (cu *cadvisorClient) ImagesFsInfo(context.Context) (cadvisorapiv2.FsInfo, error) {
	return cadvisorapiv2.FsInfo{}, nil
}

func (cu *cadvisorClient) ContainerFsInfo(context.Context) (cadvisorapiv2.FsInfo, error) {
	return cadvisorapiv2.FsInfo{}, nil
}

func (cu *cadvisorClient) RootFsInfo() (cadvisorapiv2.FsInfo, error) {
	return cu.GetDirFsInfo(cu.rootPath)
}

func (cu *cadvisorClient) GetDirFsInfo(path string) (cadvisorapiv2.FsInfo, error) {
	return aixstats.DirFsInfo(path)
}

func IsPsiEnabled(_ klog.Logger) bool {
	return false
}

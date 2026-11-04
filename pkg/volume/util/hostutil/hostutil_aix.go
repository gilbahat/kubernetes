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

package hostutil

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"k8s.io/mount-utils"
	utilpath "k8s.io/utils/path"
)

// HostUtil implements HostUtils on AIX.
//
// Path and ownership queries use the same portable code as Linux. Queries
// that need the mount table or device usage are not implemented yet, because
// mount-utils has no AIX mounter.
type HostUtil struct{}

// NewHostUtil returns a struct that implements HostUtils on AIX.
func NewHostUtil() *HostUtil {
	return &HostUtil{}
}

var errUnsupported = errors.New("volume/util/hostutil: not yet implemented on aix")

// DeviceOpened is not implemented on AIX yet.
func (hu *HostUtil) DeviceOpened(pathname string) (bool, error) {
	return false, errUnsupported
}

// PathIsDevice checks whether the path is a block or character device.
func (hu *HostUtil) PathIsDevice(pathname string) (bool, error) {
	pathType, err := hu.GetFileType(pathname)
	isDevice := pathType == FileTypeCharDev || pathType == FileTypeBlockDev
	return isDevice, err
}

// GetDeviceNameFromMount is not implemented on AIX yet, because it needs the
// mount table.
func (hu *HostUtil) GetDeviceNameFromMount(mounter mount.Interface, mountPath, pluginMountDir string) (string, error) {
	return getDeviceNameFromMount(mounter, mountPath, pluginMountDir)
}

func getDeviceNameFromMount(mounter mount.Interface, mountPath, pluginMountDir string) (string, error) {
	return "", errUnsupported
}

// MakeRShared is a no-op on AIX. AIX has no mount propagation (shared
// subtrees), so there is nothing to configure; the kubelet calls this on its
// root directory at startup.
func (hu *HostUtil) MakeRShared(path string) error {
	return nil
}

// GetFileType checks for file, directory, socket, block and character devices.
func (hu *HostUtil) GetFileType(pathname string) (FileType, error) {
	return getFileType(pathname)
}

// PathExists checks whether the path exists, following symlinks.
func (hu *HostUtil) PathExists(pathname string) (bool, error) {
	return utilpath.Exists(utilpath.CheckFollowSymlink, pathname)
}

// EvalHostSymlinks returns the path name after evaluating symlinks.
func (hu *HostUtil) EvalHostSymlinks(pathname string) (string, error) {
	return filepath.EvalSymlinks(pathname)
}

// GetOwner returns the UID and GID of the path, after evaluating symlinks.
func (hu *HostUtil) GetOwner(pathname string) (int64, int64, error) {
	realpath, err := filepath.EvalSymlinks(pathname)
	if err != nil {
		return -1, -1, err
	}
	info, err := os.Stat(realpath)
	if err != nil {
		return -1, -1, err
	}
	stat := info.Sys().(*syscall.Stat_t)
	return int64(stat.Uid), int64(stat.Gid), nil
}

// GetSELinuxSupport returns false: AIX has no SELinux.
func (hu *HostUtil) GetSELinuxSupport(pathname string) (bool, error) {
	return false, nil
}

// GetMode returns the permissions of the path.
func (hu *HostUtil) GetMode(pathname string) (os.FileMode, error) {
	info, err := os.Stat(pathname)
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}

// GetSELinuxMountContext returns an empty context: AIX has no SELinux.
func (hu *HostUtil) GetSELinuxMountContext(pathname string) (string, error) {
	return "", nil
}

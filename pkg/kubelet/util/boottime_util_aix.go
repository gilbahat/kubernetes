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

package util

import (
	"fmt"
	"time"
)

// GetBootTime is not implemented on AIX yet. Callers tolerate the error:
// the stats summary reports a zero node start time and the startup latency
// tracker skips the boot-to-kubelet interval.
//
// TODO(aix): read the boot time from the BOOT_TIME record in /etc/utmp or
// from libperfstat once it can be verified on AIX hardware.
func GetBootTime() (time.Time, error) {
	return time.Time{}, fmt.Errorf("GetBootTime is not yet implemented on AIX")
}

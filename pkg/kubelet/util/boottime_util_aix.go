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
	"time"

	"k8s.io/kubernetes/pkg/kubelet/aixstats"
)

// GetBootTime returns the time at which the machine was started, to the
// minute, from the boot record that `who -b` reads from /etc/utmp.
func GetBootTime() (time.Time, error) {
	return aixstats.BootTime()
}

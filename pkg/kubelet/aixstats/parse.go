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

// Package aixstats reads node information on AIX for the kubelet's cAdvisor
// implementation, the way package winstats does on Windows.
//
// The parsers in this file are platform independent so they can be unit
// tested anywhere; the code that calls AIX syscalls and commands is in
// aixstats_aix.go.
package aixstats

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

// parseWhoBoot parses the output of `who -b` into the boot time.
//
// The command must run with LC_ALL=C and TZ=UTC0, so that month names are in
// English and times are in UTC. AIX prints the boot record without a year,
// for example "   .        system boot  Nov 04 07:12"; the year is taken
// from now, moving back one year if that would put the boot in the future.
// GNU who (used in tests) prints "         system boot  2026-11-04 07:12".
func parseWhoBoot(out string, now time.Time) (time.Time, error) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f != "boot" || i == 0 || fields[i-1] != "system" {
				continue
			}
			rest := fields[i+1:]
			if len(rest) >= 2 {
				if t, err := time.Parse("2006-01-02 15:04", rest[0]+" "+rest[1]); err == nil {
					return t, nil
				}
			}
			if len(rest) >= 3 {
				t, err := time.Parse("Jan 2 15:04", rest[0]+" "+rest[1]+" "+rest[2])
				if err != nil {
					return time.Time{}, fmt.Errorf("unrecognised boot time %q: %w", strings.Join(rest, " "), err)
				}
				nowUTC := now.UTC()
				t = time.Date(nowUTC.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)
				if t.After(nowUTC.Add(24 * time.Hour)) {
					t = t.AddDate(-1, 0, 0)
				}
				return t, nil
			}
			return time.Time{}, fmt.Errorf("no boot time after %q in %q", "system boot", line)
		}
	}
	return time.Time{}, fmt.Errorf("no %q record in who output %q", "system boot", out)
}

// bootIDFromTime derives a stable, UUID-shaped boot ID from the boot time.
// AIX has no per-boot random ID like Linux's
// /proc/sys/kernel/random/boot_id. The boot time recorded in /etc/utmp is
// stable for the life of a boot, so two kubelet runs in the same boot report
// the same ID. Two boots within the same minute would collide.
func bootIDFromTime(bootTime time.Time) string {
	sum := sha256.Sum256([]byte("aix-boot:" + bootTime.UTC().Format(time.RFC3339)))
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

// parseLsattrValue returns the attribute value printed by
// `lsattr -E -l sys0 -a <attr> -F value`, or "" if there is none.
func parseLsattrValue(out string) string {
	v := strings.TrimSpace(out)
	if strings.ContainsAny(v, " \t\n") {
		return ""
	}
	return v
}

// kernelVersion builds the AIX kernel version from uname's version and
// release fields: uname -v prints "7" and uname -r prints "3" on AIX 7.3.
func kernelVersion(version, release string) string {
	version = strings.TrimSpace(version)
	release = strings.TrimSpace(release)
	if version == "" {
		return release
	}
	if release == "" {
		return version
	}
	return version + "." + release
}

// osImage builds the node's OS image string, for example
// "AIX 7.3 (7300-02-01-2346)". osLevel is the output of `oslevel -s` and may
// be empty.
func osImage(kernelVersion, osLevel string) string {
	image := strings.TrimSpace("AIX " + kernelVersion)
	if osLevel = strings.TrimSpace(osLevel); osLevel != "" {
		image += " (" + osLevel + ")"
	}
	return image
}

// cstring returns the NUL-terminated string at the start of b.
func cstring(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

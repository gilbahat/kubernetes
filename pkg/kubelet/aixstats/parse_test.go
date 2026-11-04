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
	"regexp"
	"testing"
	"time"
)

func TestParseWhoBoot(t *testing.T) {
	now := time.Date(2026, time.November, 4, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		out     string
		want    time.Time
		wantErr bool
	}{
		{
			name: "aix format",
			out:  "   .        system boot  Nov 04 07:12\n",
			want: time.Date(2026, time.November, 4, 7, 12, 0, 0, time.UTC),
		},
		{
			name: "aix format, single space",
			out:  "   .        system boot Jun 29 10:22",
			want: time.Date(2026, time.June, 29, 10, 22, 0, 0, time.UTC),
		},
		{
			name: "aix format, booted last year",
			out:  "   .        system boot  Dec 30 23:59\n",
			want: time.Date(2025, time.December, 30, 23, 59, 0, 0, time.UTC),
		},
		{
			name: "aix format, up to a day in the future (clock skew) stays this year",
			out:  "   .        system boot  Nov 05 01:00\n",
			want: time.Date(2026, time.November, 5, 1, 0, 0, 0, time.UTC),
		},
		{
			name: "gnu format",
			out:  "         system boot  2026-11-04 07:12\n",
			want: time.Date(2026, time.November, 4, 7, 12, 0, 0, time.UTC),
		},
		{
			name:    "no boot record",
			out:     "root     pts/0        Nov 04 08:00\n",
			wantErr: true,
		},
		{
			name:    "empty",
			out:     "",
			wantErr: true,
		},
		{
			name:    "garbled time",
			out:     "   .        system boot  Xyz 04 07:12\n",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseWhoBoot(tc.out, now)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBootIDFromTime(t *testing.T) {
	boot := time.Date(2026, time.November, 4, 7, 12, 0, 0, time.UTC)
	id := bootIDFromTime(boot)
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(id) {
		t.Errorf("boot ID %q is not UUID-shaped", id)
	}
	if again := bootIDFromTime(boot.In(time.FixedZone("IST", 2*3600))); again != id {
		t.Errorf("boot ID depends on time zone: %q vs %q", again, id)
	}
	if other := bootIDFromTime(boot.Add(time.Minute)); other == id {
		t.Errorf("different boot times gave the same ID %q", id)
	}
}

func TestParseLsattrValue(t *testing.T) {
	for out, want := range map[string]string{
		"9b3c4b4e-6f3a-4bb2-9d7e-0a1b2c3d4e5f\n": "9b3c4b4e-6f3a-4bb2-9d7e-0a1b2c3d4e5f",
		"":                                       "",
		"\n":                                     "",
		"lsattr: 0514-528 The \"os_uuid\" attribute does not exist\n": "",
	} {
		if got := parseLsattrValue(out); got != want {
			t.Errorf("parseLsattrValue(%q) = %q, want %q", out, got, want)
		}
	}
}

func TestKernelVersionAndOSImage(t *testing.T) {
	kv := kernelVersion("7", "3\n")
	if kv != "7.3" {
		t.Errorf("kernelVersion = %q, want 7.3", kv)
	}
	if got := osImage(kv, "7300-02-01-2346\n"); got != "AIX 7.3 (7300-02-01-2346)" {
		t.Errorf("osImage = %q", got)
	}
	if got := osImage(kv, ""); got != "AIX 7.3" {
		t.Errorf("osImage without oslevel = %q", got)
	}
	if got := kernelVersion("", "3"); got != "3" {
		t.Errorf("kernelVersion without version = %q", got)
	}
}

func TestCstring(t *testing.T) {
	if got := cstring([]byte{'A', 'I', 'X', 0, 'x'}); got != "AIX" {
		t.Errorf("cstring = %q", got)
	}
	if got := cstring([]byte("AIX")); got != "AIX" {
		t.Errorf("cstring without NUL = %q", got)
	}
}

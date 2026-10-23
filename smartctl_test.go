// Copyright 2024 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestBuildDeviceLabel(t *testing.T) {
	tests := []struct {
		deviceName    string
		deviceType    string
		expectedLabel string
	}{
		{"/dev/bus/0", "megaraid,1", "bus_0_megaraid_1"},
		{"/dev/sda", "auto", "sda"},
		{"/dev/disk/by-id/ata-CT500MX500SSD1_ABCDEFGHIJ", "auto", "ata-CT500MX500SSD1_ABCDEFGHIJ"},
		// Some cases extracted from smartctl docs. Are these the prettiest?
		// Probably not. Are they unique enough. Definitely.
		{"/dev/sg1", "cciss,1", "sg1_cciss_1"},
		{"/dev/bsg/sssraid0", "sssraid,0,1", "bsg_sssraid0_sssraid_0_1"},
		{"/dev/cciss/c0d0", "cciss,0", "cciss_c0d0_cciss_0"},
		{"/dev/sdb", "aacraid,1,0,4", "sdb_aacraid_1_0_4"},
		{"/dev/twl0", "3ware,1", "twl0_3ware_1"},
	}

	for _, test := range tests {
		result := buildDeviceLabel(test.deviceName, test.deviceType)
		if result != test.expectedLabel {
			t.Errorf("deviceName=%v deviceType=%v expected=%v result=%v", test.deviceName, test.deviceType, test.expectedLabel, result)
		}
	}
}

// TestPedanticRegistryCollectsAllFixtures collects metrics for every
// testdata fixture inside a pedantic registry, which fails if Collect
// emits a metric whose descriptor was not returned by Describe.
func TestPedanticRegistryCollectsAllFixtures(t *testing.T) {
	*smartctlFakeData = true
	t.Cleanup(func() { *smartctlFakeData = false })

	fixtures, err := filepath.Glob("testdata/*.json")
	if err != nil || len(fixtures) == 0 {
		t.Fatalf("no testdata fixtures found: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "debug"), 0o755); err != nil {
		t.Fatal(err)
	}

	var devices []Device
	for _, fixture := range fixtures {
		name := strings.TrimSuffix(filepath.Base(fixture), ".json")
		data, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "debug", name+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		devices = append(devices, Device{Name: name, Type: "auto", Label: name})
	}
	t.Chdir(tmp)

	collector := &SMARTctlManagerCollector{
		Devices: devices,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(collector); err != nil {
		t.Fatalf("registering collector: %v", err)
	}
	if _, err := reg.Gather(); err != nil {
		t.Errorf("gathering metrics: %v", err)
	}
}

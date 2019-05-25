/*
Copyright 2017 The Kubernetes Authors.

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

package devicemanager

import (
	"flag"
	"github.com/supremind/gpu-monitoring-tools/bindings/go/nvml"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestRank(t *testing.T) {
	var logLevel string

	klog.InitFlags(flag.CommandLine)
	flag.StringVar(&logLevel, "logLevel", "3", "test")
	flag.Lookup("v").Value.Set(logLevel)

	gpus = []*nvml.Device{&nvml.Device{UUID: "0"}, &nvml.Device{UUID: "1"}, &nvml.Device{UUID: "2"}, &nvml.Device{UUID: "3"},
		&nvml.Device{UUID: "4"}, &nvml.Device{UUID: "5"}, &nvml.Device{UUID: "6"}, &nvml.Device{UUID: "7"}}
	links = map[string](map[string]nvml.P2PLinkType){
		"0": map[string]nvml.P2PLinkType{"1": nvml.P2PLinkSingleSwitch, "2": nvml.P2PLinkHostBridge, "3": nvml.P2PLinkHostBridge,
			"4": nvml.P2PLinkCrossCPU, "5": nvml.P2PLinkCrossCPU, "6": nvml.P2PLinkCrossCPU, "7": nvml.P2PLinkCrossCPU},
		"1": map[string]nvml.P2PLinkType{"0": nvml.P2PLinkSingleSwitch, "2": nvml.P2PLinkHostBridge, "3": nvml.P2PLinkHostBridge,
			"4": nvml.P2PLinkCrossCPU, "5": nvml.P2PLinkCrossCPU, "6": nvml.P2PLinkCrossCPU, "7": nvml.P2PLinkCrossCPU},
		"2": map[string]nvml.P2PLinkType{"3": nvml.P2PLinkSingleSwitch, "0": nvml.P2PLinkHostBridge, "1": nvml.P2PLinkHostBridge,
			"4": nvml.P2PLinkCrossCPU, "5": nvml.P2PLinkCrossCPU, "6": nvml.P2PLinkCrossCPU, "7": nvml.P2PLinkCrossCPU},
		"3": map[string]nvml.P2PLinkType{"2": nvml.P2PLinkSingleSwitch, "0": nvml.P2PLinkHostBridge, "1": nvml.P2PLinkHostBridge,
			"4": nvml.P2PLinkCrossCPU, "5": nvml.P2PLinkCrossCPU, "6": nvml.P2PLinkCrossCPU, "7": nvml.P2PLinkCrossCPU},
		"4": map[string]nvml.P2PLinkType{"5": nvml.P2PLinkSingleSwitch, "6": nvml.P2PLinkHostBridge, "7": nvml.P2PLinkHostBridge,
			"0": nvml.P2PLinkCrossCPU, "1": nvml.P2PLinkCrossCPU, "2": nvml.P2PLinkCrossCPU, "3": nvml.P2PLinkCrossCPU},
		"5": map[string]nvml.P2PLinkType{"4": nvml.P2PLinkSingleSwitch, "6": nvml.P2PLinkHostBridge, "7": nvml.P2PLinkHostBridge,
			"0": nvml.P2PLinkCrossCPU, "1": nvml.P2PLinkCrossCPU, "2": nvml.P2PLinkCrossCPU, "3": nvml.P2PLinkCrossCPU},
		"6": map[string]nvml.P2PLinkType{"7": nvml.P2PLinkSingleSwitch, "4": nvml.P2PLinkHostBridge, "5": nvml.P2PLinkHostBridge,
			"0": nvml.P2PLinkCrossCPU, "1": nvml.P2PLinkCrossCPU, "2": nvml.P2PLinkCrossCPU, "3": nvml.P2PLinkCrossCPU},
		"7": map[string]nvml.P2PLinkType{"6": nvml.P2PLinkSingleSwitch, "4": nvml.P2PLinkHostBridge, "5": nvml.P2PLinkHostBridge,
			"0": nvml.P2PLinkCrossCPU, "1": nvml.P2PLinkCrossCPU, "2": nvml.P2PLinkCrossCPU, "3": nvml.P2PLinkCrossCPU},
	}
	costs = map[nvml.P2PLinkType]int{nvml.P2PLinkSameBoard: 0, nvml.P2PLinkSingleSwitch: 10, nvml.P2PLinkMultiSwitch: 50,
		nvml.P2PLinkHostBridge: 100, nvml.P2PLinkSameCPU: 200, nvml.P2PLinkCrossCPU: 500}

	if myTest([]int{0, 1, 2, 3, 4, 5, 6, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 1, 2, 3, 4, 5, 6, 7}, 2) != result([]int{0, 1}) ||
		myTest([]int{0, 1, 2, 3, 4, 5, 6, 7}, 3) != result([]int{0, 1, 2}) ||
		myTest([]int{0, 1, 2, 3, 4, 5, 6, 7}, 4) != result([]int{0, 1, 2, 3}) ||
		myTest([]int{0, 1, 2, 3, 4, 5, 6, 7}, 5) != result([]int{0, 1, 2, 3, 4}) ||
		myTest([]int{0, 1, 2, 3, 4, 5, 6, 7}, 6) != result([]int{0, 1, 2, 3, 4, 5}) ||
		myTest([]int{0, 1, 2, 3, 4, 5, 6, 7}, 7) != result([]int{0, 1, 2, 3, 4, 5, 6}) ||
		myTest([]int{0, 1, 2, 3, 4, 5, 6, 7}, 8) != result([]int{0, 1, 2, 3, 4, 5, 6, 7}) ||
		myTest([]int{0, 3, 4, 5, 6, 7}, 4) != result([]int{4, 5, 6, 7}) ||
		myTest([]int{0, 3, 4, 5, 6, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 2, 4, 5, 6, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 3, 4, 5, 6, 7}, 2) != result([]int{4, 5}) ||
		myTest([]int{0, 2, 4, 5, 6, 7}, 2) != result([]int{4, 5}) ||
		myTest([]int{0, 2, 3, 4, 5, 6, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 2, 3, 4, 5, 6, 7}, 2) != result([]int{2, 3}) ||
		myTest([]int{0, 2, 3, 4, 5, 6, 7}, 3) != result([]int{0, 2, 3}) ||
		myTest([]int{0, 4, 5, 6, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 4, 5, 6, 7}, 3) != result([]int{4, 5, 6}) ||
		myTest([]int{0, 4, 5, 6, 7}, 4) != result([]int{4, 5, 6, 7}) ||
		myTest([]int{0, 4, 5, 6, 7}, 5) != result([]int{0, 4, 5, 6, 7}) ||
		myTest([]int{0, 2, 6, 7}, 4) != result([]int{0, 2, 6, 7}) ||
		myTest([]int{0, 2, 6, 7}, 2) != result([]int{6, 7}) ||
		myTest([]int{0, 1, 6, 7}, 2) != result([]int{0, 1}) ||
		myTest([]int{0, 2, 5, 7}, 4) != result([]int{0, 2, 5, 7}) ||
		myTest([]int{0, 2, 3, 7}, 3) != result([]int{0, 2, 3}) ||
		myTest([]int{0, 2, 4}, 2) != result([]int{0, 2}) ||
		myTest([]int{0, 5, 6, 7}, 3) != result([]int{5, 6, 7}) ||
		myTest([]int{0, 4, 5, 6, 7}, 6) != result([]int{}) ||
		myTest([]int{0, 2, 4}, 1) != result([]int{4}) ||
		myTest([]int{0, 2, 5}, 1) != result([]int{5}) ||
		myTest([]int{0, 2, 6}, 1) != result([]int{6}) ||
		myTest([]int{0, 2, 7}, 1) != result([]int{7}) ||
		myTest([]int{0, 3, 4}, 1) != result([]int{4}) ||
		myTest([]int{0, 3, 5}, 1) != result([]int{5}) ||
		myTest([]int{0, 3, 6}, 1) != result([]int{6}) ||
		myTest([]int{0, 3, 7}, 1) != result([]int{7}) ||
		myTest([]int{0, 4, 6}, 1) != result([]int{0}) ||
		myTest([]int{0, 4, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 5, 6}, 1) != result([]int{0}) ||
		myTest([]int{0, 5, 7}, 1) != result([]int{0}) ||
		myTest([]int{1, 2, 4}, 1) != result([]int{4}) ||
		myTest([]int{1, 2, 5}, 1) != result([]int{5}) ||
		myTest([]int{1, 2, 6}, 1) != result([]int{6}) ||
		myTest([]int{1, 2, 7}, 1) != result([]int{7}) ||
		myTest([]int{1, 3, 4}, 1) != result([]int{4}) ||
		myTest([]int{1, 3, 5}, 1) != result([]int{5}) ||
		myTest([]int{1, 3, 6}, 1) != result([]int{6}) ||
		myTest([]int{1, 3, 7}, 1) != result([]int{7}) ||
		myTest([]int{1, 4, 6}, 1) != result([]int{1}) ||
		myTest([]int{1, 4, 7}, 1) != result([]int{1}) ||
		myTest([]int{1, 5, 6}, 1) != result([]int{1}) ||
		myTest([]int{1, 5, 7}, 1) != result([]int{1}) ||
		myTest([]int{2, 4, 6}, 1) != result([]int{2}) ||
		myTest([]int{2, 4, 7}, 1) != result([]int{2}) ||
		myTest([]int{2, 5, 6}, 1) != result([]int{2}) ||
		myTest([]int{2, 5, 7}, 1) != result([]int{2}) ||
		myTest([]int{3, 4, 6}, 1) != result([]int{3}) ||
		myTest([]int{3, 4, 7}, 1) != result([]int{3}) ||
		myTest([]int{3, 5, 6}, 1) != result([]int{3}) ||
		myTest([]int{3, 5, 7}, 1) != result([]int{3}) ||
		myTest([]int{0, 1, 2, 4}, 1) != result([]int{4}) ||
		myTest([]int{0, 1, 2, 5}, 1) != result([]int{5}) ||
		myTest([]int{0, 1, 2, 6}, 1) != result([]int{6}) ||
		myTest([]int{0, 1, 2, 7}, 1) != result([]int{7}) ||
		myTest([]int{0, 1, 3, 4}, 1) != result([]int{4}) ||
		myTest([]int{0, 1, 3, 5}, 1) != result([]int{5}) ||
		myTest([]int{0, 1, 3, 6}, 1) != result([]int{6}) ||
		myTest([]int{0, 1, 3, 7}, 1) != result([]int{7}) ||
		myTest([]int{0, 2, 3, 4}, 1) != result([]int{4}) ||
		myTest([]int{0, 2, 3, 5}, 1) != result([]int{5}) ||
		myTest([]int{0, 2, 3, 6}, 1) != result([]int{6}) ||
		myTest([]int{0, 2, 3, 7}, 1) != result([]int{7}) ||
		myTest([]int{0, 4, 5, 6}, 1) != result([]int{0}) ||
		myTest([]int{0, 4, 5, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 4, 6, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 5, 6, 7}, 1) != result([]int{0}) ||
		myTest([]int{1, 2, 3, 4}, 1) != result([]int{4}) ||
		myTest([]int{1, 2, 3, 5}, 1) != result([]int{5}) ||
		myTest([]int{1, 2, 3, 6}, 1) != result([]int{6}) ||
		myTest([]int{1, 2, 3, 7}, 1) != result([]int{7}) ||
		myTest([]int{1, 4, 5, 6}, 1) != result([]int{1}) ||
		myTest([]int{1, 4, 5, 7}, 1) != result([]int{1}) ||
		myTest([]int{1, 4, 6, 7}, 1) != result([]int{1}) ||
		myTest([]int{1, 5, 6, 7}, 1) != result([]int{1}) ||
		myTest([]int{2, 4, 5, 6}, 1) != result([]int{2}) ||
		myTest([]int{2, 4, 5, 7}, 1) != result([]int{2}) ||
		myTest([]int{2, 4, 6, 7}, 1) != result([]int{2}) ||
		myTest([]int{2, 5, 6, 7}, 1) != result([]int{2}) ||
		myTest([]int{3, 4, 5, 6}, 1) != result([]int{3}) ||
		myTest([]int{3, 4, 5, 7}, 1) != result([]int{3}) ||
		myTest([]int{3, 4, 6, 7}, 1) != result([]int{3}) ||
		myTest([]int{3, 5, 6, 7}, 1) != result([]int{3}) ||
		myTest([]int{0, 1, 2, 4, 5}, 1) != result([]int{2}) ||
		myTest([]int{0, 1, 2, 4, 5}, 2) != result([]int{4, 5}) ||
		myTest([]int{0, 1, 2, 4, 5}, 4) != result([]int{0, 1, 4, 5}) ||
		myTest([]int{0, 1, 2, 4, 6}, 4) != result([]int{0, 1, 4, 6}) ||
		myTest([]int{0, 1, 2, 4, 7}, 4) != result([]int{0, 1, 4, 7}) ||
		myTest([]int{0, 1, 2, 5, 6}, 4) != result([]int{0, 1, 5, 6}) ||
		myTest([]int{0, 1, 2, 5, 7}, 4) != result([]int{0, 1, 5, 7}) ||
		myTest([]int{0, 1, 2, 6, 7}, 1) != result([]int{2}) ||
		myTest([]int{0, 1, 2, 6, 7}, 2) != result([]int{6, 7}) ||
		myTest([]int{0, 1, 2, 6, 7}, 4) != result([]int{0, 1, 6, 7}) ||
		myTest([]int{0, 1, 3, 4, 5}, 1) != result([]int{3}) ||
		myTest([]int{0, 1, 3, 4, 5}, 2) != result([]int{4, 5}) ||
		myTest([]int{0, 1, 3, 4, 5}, 4) != result([]int{0, 1, 4, 5}) ||
		myTest([]int{0, 1, 3, 4, 6}, 4) != result([]int{0, 1, 4, 6}) ||
		myTest([]int{0, 1, 3, 4, 7}, 4) != result([]int{0, 1, 4, 7}) ||
		myTest([]int{0, 1, 3, 5, 6}, 4) != result([]int{0, 1, 5, 6}) ||
		myTest([]int{0, 1, 3, 5, 7}, 4) != result([]int{0, 1, 5, 7}) ||
		myTest([]int{0, 1, 3, 6, 7}, 1) != result([]int{3}) ||
		myTest([]int{0, 1, 3, 6, 7}, 2) != result([]int{6, 7}) ||
		myTest([]int{0, 1, 3, 6, 7}, 4) != result([]int{0, 1, 6, 7}) ||
		myTest([]int{0, 1, 4, 5, 6}, 1) != result([]int{6}) ||
		myTest([]int{0, 1, 4, 5, 6}, 2) != result([]int{0, 1}) ||
		myTest([]int{0, 1, 4, 5, 6}, 4) != result([]int{0, 1, 4, 5}) ||
		myTest([]int{0, 1, 4, 5, 7}, 1) != result([]int{7}) ||
		myTest([]int{0, 1, 4, 5, 7}, 2) != result([]int{0, 1}) ||
		myTest([]int{0, 1, 4, 5, 7}, 4) != result([]int{0, 1, 4, 5}) ||
		myTest([]int{0, 1, 4, 6, 7}, 1) != result([]int{4}) ||
		myTest([]int{0, 1, 4, 6, 7}, 2) != result([]int{0, 1}) ||
		myTest([]int{0, 1, 4, 6, 7}, 4) != result([]int{0, 1, 6, 7}) ||
		myTest([]int{0, 1, 5, 6, 7}, 1) != result([]int{5}) ||
		myTest([]int{0, 1, 5, 6, 7}, 2) != result([]int{0, 1}) ||
		myTest([]int{0, 1, 5, 6, 7}, 4) != result([]int{0, 1, 6, 7}) ||
		myTest([]int{0, 2, 3, 4, 5}, 1) != result([]int{0}) ||
		myTest([]int{0, 2, 3, 4, 5}, 2) != result([]int{4, 5}) ||
		myTest([]int{0, 2, 3, 4, 5}, 4) != result([]int{2, 3, 4, 5}) ||
		myTest([]int{0, 2, 3, 4, 6}, 4) != result([]int{2, 3, 4, 6}) ||
		myTest([]int{0, 2, 3, 4, 7}, 4) != result([]int{2, 3, 4, 7}) ||
		myTest([]int{0, 2, 3, 5, 6}, 4) != result([]int{2, 3, 5, 6}) ||
		myTest([]int{0, 2, 3, 5, 7}, 4) != result([]int{2, 3, 5, 7}) ||
		myTest([]int{0, 2, 3, 6, 7}, 1) != result([]int{0}) ||
		myTest([]int{0, 2, 3, 6, 7}, 2) != result([]int{6, 7}) ||
		myTest([]int{0, 2, 3, 6, 7}, 4) != result([]int{2, 3, 6, 7}) ||
		myTest([]int{0, 2, 4, 5, 6}, 4) != result([]int{0, 2, 4, 5}) ||
		myTest([]int{0, 2, 4, 5, 7}, 4) != result([]int{4, 5, 0, 2}) ||
		myTest([]int{0, 2, 4, 6, 7}, 4) != result([]int{6, 7, 0, 2}) ||
		myTest([]int{0, 2, 5, 6, 7}, 4) != result([]int{6, 7, 0, 2}) ||
		myTest([]int{0, 3, 4, 5, 6}, 4) != result([]int{4, 5, 0, 3}) ||
		myTest([]int{0, 3, 4, 5, 7}, 4) != result([]int{4, 5, 0, 3}) ||
		myTest([]int{0, 3, 4, 6, 7}, 4) != result([]int{6, 7, 0, 3}) ||
		myTest([]int{0, 3, 5, 6, 7}, 4) != result([]int{6, 7, 0, 3}) ||
		myTest([]int{1, 2, 3, 4, 5}, 1) != result([]int{1}) ||
		myTest([]int{1, 2, 3, 4, 5}, 2) != result([]int{4, 5}) ||
		myTest([]int{1, 2, 3, 4, 5}, 4) != result([]int{4, 5, 2, 3}) ||
		myTest([]int{1, 2, 3, 4, 6}, 4) != result([]int{2, 3, 4, 6}) ||
		myTest([]int{1, 2, 3, 4, 7}, 4) != result([]int{2, 3, 4, 7}) ||
		myTest([]int{1, 2, 3, 5, 6}, 4) != result([]int{2, 3, 5, 6}) ||
		myTest([]int{1, 2, 3, 5, 7}, 4) != result([]int{2, 3, 5, 7}) ||
		myTest([]int{1, 2, 3, 6, 7}, 1) != result([]int{1}) ||
		myTest([]int{1, 2, 3, 6, 7}, 2) != result([]int{6, 7}) ||
		myTest([]int{1, 2, 3, 6, 7}, 4) != result([]int{6, 7, 2, 3}) ||
		myTest([]int{1, 2, 4, 5, 6}, 4) != result([]int{4, 5, 1, 2}) ||
		myTest([]int{1, 2, 4, 5, 7}, 4) != result([]int{4, 5, 1, 2}) ||
		myTest([]int{1, 2, 4, 6, 7}, 4) != result([]int{6, 7, 1, 2}) ||
		myTest([]int{1, 2, 5, 6, 7}, 4) != result([]int{6, 7, 1, 2}) ||
		myTest([]int{1, 3, 4, 5, 6}, 4) != result([]int{4, 5, 1, 3}) ||
		myTest([]int{1, 3, 4, 5, 7}, 4) != result([]int{4, 5, 1, 3}) ||
		myTest([]int{1, 3, 4, 6, 7}, 4) != result([]int{6, 7, 1, 3}) ||
		myTest([]int{1, 3, 5, 6, 7}, 4) != result([]int{6, 7, 1, 3}) ||
		myTest([]int{2, 3, 4, 5, 6}, 1) != result([]int{6}) ||
		myTest([]int{2, 3, 4, 5, 6}, 2) != result([]int{2, 3}) ||
		myTest([]int{2, 3, 4, 5, 6}, 4) != result([]int{2, 3, 4, 5}) ||
		myTest([]int{2, 3, 4, 5, 7}, 1) != result([]int{7}) ||
		myTest([]int{2, 3, 4, 5, 7}, 2) != result([]int{2, 3}) ||
		myTest([]int{2, 3, 4, 5, 7}, 4) != result([]int{2, 3, 4, 5}) ||
		myTest([]int{2, 3, 4, 6, 7}, 1) != result([]int{4}) ||
		myTest([]int{2, 3, 4, 6, 7}, 2) != result([]int{2, 3}) ||
		myTest([]int{2, 3, 4, 6, 7}, 4) != result([]int{2, 3, 6, 7}) ||
		myTest([]int{2, 3, 5, 6, 7}, 1) != result([]int{5}) ||
		myTest([]int{2, 3, 5, 6, 7}, 2) != result([]int{2, 3}) ||
		myTest([]int{2, 3, 5, 6, 7}, 4) != result([]int{2, 3, 6, 7}) ||
		myTest([]int{0, 1, 2, 3, 4, 5}, 2) != result([]int{4, 5}) ||
		myTest([]int{0, 1, 2, 3, 6, 7}, 2) != result([]int{6, 7}) ||
		myTest([]int{0, 1, 2, 4, 5, 6}, 4) != result([]int{0, 1, 4, 5}) ||
		myTest([]int{0, 1, 2, 4, 5, 7}, 4) != result([]int{0, 1, 4, 5}) ||
		myTest([]int{0, 1, 2, 4, 6, 7}, 4) != result([]int{0, 1, 6, 7}) ||
		myTest([]int{0, 1, 2, 5, 6, 7}, 4) != result([]int{0, 1, 6, 7}) ||
		myTest([]int{0, 1, 3, 4, 5, 6}, 4) != result([]int{0, 1, 4, 5}) ||
		myTest([]int{0, 1, 3, 4, 5, 7}, 4) != result([]int{0, 1, 4, 5}) ||
		myTest([]int{0, 1, 3, 4, 6, 7}, 4) != result([]int{0, 1, 6, 7}) ||
		myTest([]int{0, 1, 3, 5, 6, 7}, 4) != result([]int{0, 1, 6, 7}) ||
		myTest([]int{0, 1, 4, 5, 6, 7}, 2) != result([]int{0, 1}) ||
		myTest([]int{0, 2, 3, 4, 5, 6}, 4) != result([]int{2, 3, 4, 5}) ||
		myTest([]int{0, 2, 3, 4, 5, 7}, 4) != result([]int{2, 3, 4, 5}) ||
		myTest([]int{0, 2, 3, 4, 6, 7}, 4) != result([]int{2, 3, 6, 7}) ||
		myTest([]int{0, 2, 3, 5, 6, 7}, 4) != result([]int{2, 3, 6, 7}) ||
		myTest([]int{1, 2, 3, 4, 5, 6}, 4) != result([]int{2, 3, 4, 5}) ||
		myTest([]int{1, 2, 3, 4, 5, 7}, 4) != result([]int{2, 3, 4, 5}) ||
		myTest([]int{1, 2, 3, 4, 6, 7}, 4) != result([]int{2, 3, 6, 7}) ||
		myTest([]int{1, 2, 3, 5, 6, 7}, 4) != result([]int{2, 3, 6, 7}) ||
		myTest([]int{2, 3, 4, 5, 6, 7}, 2) != result([]int{2, 3}) {
		t.FailNow()
	}
}

func r0(r []int, pr bool) string {
	ret := []string{}
	for _, i := range r {
		ret = append(ret, strconv.Itoa(i))
	}

	sort.Sort(sort.StringSlice(ret))
	str := strings.Join(ret, ", ")
	if pr {
		klog.V(2).Infoln(str)
	}
	return str
}

func result(r []int) string {
	return r0(r, true)
}

func myTest(avail []int, need int) string {
	available := make(sets.String, 0)
	for _, i := range avail {
		available.Insert(strconv.Itoa(i))
	}
	ret := calcAllocated(nvidiaGPU, need, sets.String{}, available)
	sort.Sort(sort.StringSlice(ret))
	str := strings.Join(ret, ", ")
	klog.V(2).Info(r0(avail, false), " : ", need, " -> ", str)
	return str
}

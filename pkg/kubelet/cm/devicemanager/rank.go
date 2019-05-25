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
	"github.com/supremind/gpu-monitoring-tools/bindings/go/nvml"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"sort"
)

const (
	STATE_NONE  = 0
	STATE_AVAIL = 1
	STATE_INUSE = 2
	MAXCOST     = 100000
	nvidiaGPU   = "nvidia.com/gpu"
)

var stateName = map[int]string{STATE_NONE: "none", STATE_AVAIL: "avail", STATE_INUSE: "used"}

var gpus []*nvml.Device
var links map[string](map[string]nvml.P2PLinkType)
var costs map[nvml.P2PLinkType]int

// align to exponential of 2
func align2(need int) int {
	if need <= 1 {
		return need
	}
	i := 1
	for i <= need {
		i = 2 * i
	}
	return i / 2
}

// load topo from nvml
func setupRank() bool {
	if costs != nil {
		return true
	}
	nvml.Init()
	defer nvml.Shutdown()

	gpus, links, costs = nil, nil, nil
	count, err := nvml.GetDeviceCount()
	if err != nil {
		return false
	}
	for i := uint(0); i < count; i++ {
		device, err := nvml.NewDevice(i)
		if err != nil {
			return false
		}
		gpus = append(gpus, device)
	}
	for i := uint(0); i < count; i++ {
		for j := i + 1; j < count; j++ {
			if link, err := nvml.GetP2PLink(gpus[i], gpus[j]); err == nil {
				if _, ok := links[gpus[i].UUID]; !ok {
					links[gpus[i].UUID] = map[string]nvml.P2PLinkType{}
				}
				if _, ok := links[gpus[j].UUID]; !ok {
					links[gpus[j].UUID] = map[string]nvml.P2PLinkType{}
				}
				links[gpus[i].UUID][gpus[j].UUID] = link
				links[gpus[j].UUID][gpus[i].UUID] = link
			} else {
				return false
			}
		}
	}

	costs = map[nvml.P2PLinkType]int{nvml.P2PLinkSameBoard: 0, nvml.P2PLinkSingleSwitch: 10, nvml.P2PLinkMultiSwitch: 50,
		nvml.P2PLinkHostBridge: 100, nvml.P2PLinkSameCPU: 200, nvml.P2PLinkCrossCPU: 500}
	return true
}

type node struct {
	parent   *node
	children []*node
	uuid     string
	state    int
	link     nvml.P2PLinkType
}

// first common parent's link type
func p2p(l, r *node) nvml.P2PLinkType {
	var lp, rp []*node
	for p := l.parent; p != nil; p = p.parent {
		lp = append(lp, p)
	}
	for p := r.parent; p != nil; p = p.parent {
		rp = append(rp, p)
	}
	for _, p1 := range lp {
		for _, p2 := range rp {
			if p1 == p2 {
				return p1.link
			}
		}
	}
	klog.Error("SMAFFINITY can't find p2p for", l.uuid, r.uuid)
	return nvml.P2PLinkCrossCPU // should not go here
}

// return root node & inused node list
func mkNode(inuse, available sets.String) (*node, []*node) {
	nodes, usedev := []*node{}, []*node{}
	for _, d := range gpus {
		state := STATE_NONE
		if available.Has(d.UUID) {
			state = STATE_AVAIL
		}
		n := &node{uuid: d.UUID, state: state, link: nvml.P2PLinkSameBoard}
		nodes = append(nodes, n)
		if inuse.Has(d.UUID) {
			usedev = append(usedev, n)
		}
	}

	for _, link := range []nvml.P2PLinkType{nvml.P2PLinkSameBoard, nvml.P2PLinkSingleSwitch, nvml.P2PLinkMultiSwitch,
		nvml.P2PLinkHostBridge, nvml.P2PLinkSameCPU, nvml.P2PLinkCrossCPU} {
		for i := 0; i < len(nodes); i++ {
			if nodes[i] == nil {
				continue
			}
			for j := i + 1; j < len(nodes); j++ {
				if nodes[j] == nil || links[nodes[i].uuid][nodes[j].uuid] != link {
					continue
				}
				klog.V(2).Infoln("SMAFFINITY merge", nodes[i].names(STATE_NONE), nodes[j].names(STATE_NONE), link)
				if nodes[i].link != link { // create and merge into new parent node
					nodes[i] = &node{children: []*node{nodes[i]}, uuid: nodes[i].uuid, state: STATE_NONE, link: link}
					nodes[i].children[0].parent = nodes[i]
				}
				nodes[i].children = append(nodes[i].children, nodes[j]) // merge into parent node
				nodes[j].parent = nodes[i]
				nodes[j] = nil
			}
		}
	}

	return nodes[0], usedev
}

// devices in this tree
func (n *node) devices(state int) []*node {
	if len(n.children) == 0 { // leaf
		if state == STATE_NONE || n.state == state {
			return []*node{n}
		} else {
			return []*node{}
		}
	}
	avail := []*node{}
	for _, c := range n.children {
		avail = append(avail, c.devices(state)...)
	}
	return avail
}

//debug
func (n *node) names(state int) []string {
	name := []string{}
	for _, nm := range n.devices(state) {
		name = append(name, nm.uuid+":"+stateName[nm.state])
	}
	return name
}

// sum of communication cost between available & just-allocated resources in this node,
func (n *node) cost(inuse []*node) int {
	nodes := append([]*node{}, inuse...)
	nodes = append(nodes, n.devices(STATE_AVAIL)...)
	c := 0
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			c += costs[p2p(nodes[i], nodes[j])]
		}
	}
	return c
}

func cost(n *node, inuse []*node) int {
	if n == nil {
		return MAXCOST
	}
	return n.cost(inuse)
}

// step1: if only have *num* resources, rank this node and return
// step2: rank children, select which one having less cost
// step3: in case left cost = right cost, checking parent's cost recursively
// step4: left & right can not satisfy, go back and come again with half the resources
func (n *node) rank(num int, inuse []*node) *node {
	if num > len(n.devices(STATE_AVAIL)) {
		return nil
	} else if len(n.devices(STATE_AVAIL)) == num {
		return n
	}

	var nodes []*node
	for _, c := range n.children {
		if rc := c.rank(num, inuse); rc != nil {
			nodes = append(nodes, rc)
			klog.V(3).Infoln("SMAFFINITY select", "parent:", n.names(STATE_NONE), n.link, "child:", c.names(STATE_NONE), c.link,
				"to:", rc.names(STATE_NONE), rc.link, "for req", num)
		}
	}
	if len(nodes) == 0 {
		return nil
	}

	// sort step1: select node with less cost
	//      step1.1: if node cost equals, continue check parent's cost until reaching root
	// sort step2: if cost equals, select one closer to root, which implies more resourced allocated
	// sort step3: else, select left part (sort lambda i always > j)
	sort.Slice(nodes, func(i, j int) bool {
		ni, nj, li, lj := nodes[i], nodes[j], nodes[i].link, nodes[j].link
		ci, cj := 0, 0
		for ; ni != nj; ni, nj = ni.parent, nj.parent {
			if ci, cj = cost(ni, inuse), cost(nj, inuse); ci != cj {
				break
			}
			if li, lj = ni.link, nj.link; li != lj {
				break
			}
		}
		return (ci < cj) || (ci == cj && li < lj) || (li == lj && i < j)
	})

	return nodes[0]
}

// break into integral parts and do ranking separately.
// for example, for needed=7, rank 4 + 2 + 1 separately
func (root *node) allocate(needed int, inuse []*node) []string {
	ret := []string{}
	if len(root.devices(STATE_AVAIL)) < needed {
		klog.Error("SMAFFINITY wrong: no resource for", needed, root)
		return ret
	}
	num := align2(needed)
	for needed > 0 && num > 0 { // num should always > 0
		n := root.rank(num, inuse)
		if n == nil {
			klog.V(3).Infoln("SMAFFINITY will cont with half for", num)
			num = num / 2
			continue // half again
		}
		for _, dev := range n.devices(STATE_AVAIL) {
			dev.state = STATE_INUSE
			inuse = append(inuse, dev)
			ret = append(ret, dev.uuid)
			klog.V(3).Infoln("SMAFFINITY found uuid", dev.uuid, "for req", num)
		}
		needed = needed - num
		num = align2(needed)
	}
	return ret
}

// @resource: e.g nvidia.com/gpu
// @needed: how many resources to allocate
// @inuse: some pod have pre-init containers to use resources
// @available: currently available resource
func calcAllocated(resource string, needed int, inuse, available sets.String) []string {
	if resource != nvidiaGPU {
		return available.UnsortedList()[:needed]
	}

	if rc := setupRank(); !rc {
		return available.UnsortedList()[:needed]
	}

	if root, inusedevs := mkNode(inuse, available); root != nil {
		return root.allocate(needed, inusedevs)
	} else {
		klog.Error("SMAFFINITY bad algo ", resource, needed, inuse, available)
		return available.UnsortedList()[:needed]
	}
}

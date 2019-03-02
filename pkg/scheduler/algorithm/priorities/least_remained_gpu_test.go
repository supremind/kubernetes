package priorities

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/api"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
)

func TestLeastRemainedGPU(t *testing.T) {
	noResources := v1.PodSpec{
		Containers: []v1.Container{},
	}
	cpuOnly := v1.PodSpec{
		NodeName: "machine1",
		Containers: []v1.Container{
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("1000m"),
						v1.ResourceMemory: resource.MustParse("0"),
					},
				},
			},
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("2000m"),
						v1.ResourceMemory: resource.MustParse("0"),
					},
				},
			},
		},
	}
	cpuAndMemory := v1.PodSpec{
		NodeName: "machine2",
		Containers: []v1.Container{
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("1000m"),
						v1.ResourceMemory: resource.MustParse("2000"),
					},
				},
			},
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("2000m"),
						v1.ResourceMemory: resource.MustParse("3000"),
					},
				},
			},
		},
	}
	gpu1 := v1.PodSpec{
		NodeName: "machine1",
		Containers: []v1.Container{
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:        resource.MustParse("1000m"),
						v1.ResourceMemory:     resource.MustParse("0"),
						nvidiaGPUResourceName: resource.MustParse("1"),
					},
					Limits: v1.ResourceList{
						nvidiaGPUResourceName: resource.MustParse("1"),
					},
				},
			},
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("2000m"),
						v1.ResourceMemory: resource.MustParse("0"),
					},
				},
			},
		},
	}
	gpu2 := v1.PodSpec{
		NodeName: "machine2",
		Containers: []v1.Container{
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:        resource.MustParse("1000m"),
						v1.ResourceMemory:     resource.MustParse("0"),
						nvidiaGPUResourceName: resource.MustParse("1"),
					},
					Limits: v1.ResourceList{
						nvidiaGPUResourceName: resource.MustParse("1"),
					},
				},
			},
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:        resource.MustParse("2000m"),
						v1.ResourceMemory:     resource.MustParse("0"),
						nvidiaGPUResourceName: resource.MustParse("1"),
					},
					Limits: v1.ResourceList{
						nvidiaGPUResourceName: resource.MustParse("1"),
					},
				},
			},
		},
	}
	gpu3 := v1.PodSpec{
		NodeName: "machine3",
		Containers: []v1.Container{
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:        resource.MustParse("1000m"),
						v1.ResourceMemory:     resource.MustParse("0"),
						nvidiaGPUResourceName: resource.MustParse("1"),
					},
					Limits: v1.ResourceList{
						nvidiaGPUResourceName: resource.MustParse("1"),
					},
				},
			},
		},
	}
	gpu4 := v1.PodSpec{
		NodeName: "machine4",
		Containers: []v1.Container{
			{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:        resource.MustParse("1000m"),
						v1.ResourceMemory:     resource.MustParse("0"),
						nvidiaGPUResourceName: resource.MustParse("4"),
					},
					Limits: v1.ResourceList{
						nvidiaGPUResourceName: resource.MustParse("4"),
					},
				},
			},
		},
	}

	tests := []struct {
		pod          *v1.Pod
		pods         []*v1.Pod
		nodes        []*v1.Node
		expectedList schedulerapi.HostPriorityList
		test         string
	}{
		{
			pod:          &v1.Pod{Spec: noResources},
			nodes:        []*v1.Node{makeNode("machine1", 4000, 10000), makeNode("machine2", 4000, 10000)},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 0}, {Host: "machine2", Score: 0}},
			test:         "nothing scheduled, nothing requested",
		},
		{
			pod:          &v1.Pod{Spec: noResources},
			nodes:        []*v1.Node{makeGPUNode("machine1", 4000, 10000, 8, "gpu-model-1"), makeGPUNode("machine2", 4000, 10000, 8, "gpu-model-1")},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 0}, {Host: "machine2", Score: 0}},
			test:         "no gpu requested, pods scheduled with gpu",
			pods: []*v1.Pod{
				{Spec: cpuOnly},
				{Spec: cpuAndMemory},
				{Spec: gpu1},
				{Spec: gpu2},
			},
		},
		{
			/*
				Node1 scores on 0-10 scale: 10 (3 remained)
				Node2 scores on 0-10 scale:  9 (7 remained)
			*/
			pod:          &v1.Pod{Spec: gpu1},
			nodes:        []*v1.Node{makeGPUNode("machine1", 4000, 10000, 4, "gpu-model-1"), makeGPUNode("machine2", 4000, 10000, 8, "gpu-model-1")},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 97}, {Host: "machine2", Score: 93}},
			test:         "nothing scheduled, 1 gpu requested, machines with different number of gpus",
		},
		{
			pod:          &v1.Pod{Spec: gpu2},
			nodes:        []*v1.Node{makeGPUNode("machine1", 4000, 10000, 8, "gpu-model-1"), makeGPUNode("machine2", 4000, 10000, 8, "gpu-model-1")},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 96}, {Host: "machine2", Score: 98}},
			test:         "2 gpu requested, pods scheduled with different number of gpus scheduled",
			pods: []*v1.Pod{
				{Spec: gpu1},
				{Spec: gpu1},
				{Spec: gpu2},
				{Spec: gpu2},
			},
		},
		{
			pod: &v1.Pod{Spec: gpu2},
			nodes: []*v1.Node{
				makeGPUNode("machine1", 4000, 10000, 4, "gpu-model-1"),
				makeGPUNode("machine2", 4000, 10000, 4, "gpu-model-1"),
				makeGPUNode("machine3", 4000, 10000, 8, "gpu-model-1"),
				makeGPUNode("machine4", 4000, 10000, 8, "gpu-model-1"),
			},
			expectedList: []schedulerapi.HostPriority{
				{Host: "machine1", Score: 98},
				{Host: "machine2", Score: 100},
				{Host: "machine3", Score: 95},
				{Host: "machine4", Score: 98},
			},
			test: "2 gpu requested, pods scheduled with different or same number of gpus",
			pods: []*v1.Pod{
				{Spec: gpu2},
				{Spec: gpu3},
				{Spec: gpu4},
			},
		},
		{
			pod:          &v1.Pod{Spec: gpu2},
			nodes:        []*v1.Node{makeGPUNode("machine1", 4000, 10000, 4, "gpu-model-1"), makeGPUNode("machine4", 4000, 10000, 8, "gpu-model-1")},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 99}, {Host: "machine4", Score: 0}},
			test:         "requested gpu exceed node capacity",
			pods: []*v1.Pod{
				{Spec: gpu1},
				{Spec: gpu4},
				{Spec: gpu4},
			},
		},
	}

	for _, test := range tests {
		nodeNameToInfo := schedulercache.CreateNodeNameToInfoMap(test.pods, test.nodes)
		t.Log("testing: ", test.test)
		list, err := priorityFunction(LeastRemainedGPUPriorityMap, nil, nil)(test.pod, nodeNameToInfo, test.nodes)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(test.expectedList, list) {
			t.Errorf("%s: expected %#v, got %#v", test.test, test.expectedList, list)
		}
	}
}

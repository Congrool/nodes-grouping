package utils

import (
	"testing"

	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
)

func TestDesiredPodsNumInTargetNodeGroups(t *testing.T) {
	cases := []struct {
		name     string
		weights  []groupmanagementv1alpha1.StaticNodeGroupWeight
		replicas int
		want     map[string]int32
	}{
		{
			name: "normal case",
			weights: []groupmanagementv1alpha1.StaticNodeGroupWeight{
				{
					NodeGroupNames: []string{
						"beijing",
					},
					Weight: 1,
				},
				{
					NodeGroupNames: []string{
						"hangzhou",
					},
					Weight: 1,
				},
			},
			replicas: 10,
			want: map[string]int32{
				"beijing":  5,
				"hangzhou": 5,
			},
		},
		{
			name: "multi nodegroup in one weight entry",
			weights: []groupmanagementv1alpha1.StaticNodeGroupWeight{
				{
					NodeGroupNames: []string{
						"beijing",
						"hangzhou",
					},
					Weight: 1,
				},
				{
					NodeGroupNames: []string{
						"shanghai",
					},
					Weight: 4,
				},
			},
			replicas: 10,
			want: map[string]int32{
				"beijing":  2,
				"hangzhou": 0,
				"shanghai": 8,
			},
		},
		{
			name: "zero weight",
			weights: []groupmanagementv1alpha1.StaticNodeGroupWeight{
				{
					NodeGroupNames: []string{
						"beijing",
					},
					Weight: 0,
				},
				{
					NodeGroupNames: []string{
						"shanghai",
					},
					Weight: 1,
				},
			},
			replicas: 10,
			want: map[string]int32{
				"beijing":  0,
				"shanghai": 10,
			},
		},
	}

	for _, c := range cases {
		desiredPodNum := DesiredPodsNumInTargetNodeGroups(c.weights, int32(c.replicas))
		for gname, num := range desiredPodNum {
			wantnum, ok := c.want[gname]
			if !ok {
				t.Errorf("case: %s, unkown node group name: %s", c.name, gname)
				continue
			}
			if wantnum != num {
				t.Errorf("case: %s, inconsistent desired pod number, want %d but get %d", c.name, wantnum, num)
				continue
			}
		}
	}
}

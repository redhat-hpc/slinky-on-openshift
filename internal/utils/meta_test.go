// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
)

func TestKeyFunc(t *testing.T) {
	type args struct {
		obj metav1.Object
	}
	ns := &slinkyv1alpha1.NodeSet{}
	ns.SetName("nodeSetTest")
	ns.SetNamespace("slurm")
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "No namespace",
			args: args{
				obj: &slinkyv1alpha1.NodeSet{},
			},
			want: "/",
		},
		{
			name: "Slurm namespace",
			args: args{
				obj: ns,
			},
			want: "slurm/nodeSetTest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KeyFunc(tt.args.obj); got != tt.want {
				t.Errorf("KeyFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

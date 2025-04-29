// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package slurmcontrol

import (
	"errors"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	v0041 "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
	"github.com/SlinkyProject/slurm-client/pkg/types"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	"github.com/SlinkyProject/slurm-operator/internal/resources"
)

func newSlurmClusters(clusterName string, client client.Client) *resources.Clusters {
	clusters := resources.NewClusters()
	key := k8stypes.NamespacedName{
		Namespace: corev1.NamespaceDefault,
		Name:      clusterName,
	}
	clusters.Add(key, client)
	return clusters
}

var _ = Describe("SlurmControlInterface", func() {
	const clusterName string = "foo"
	var slurmcontrol SlurmControlInterface
	var cluster *slinkyv1alpha1.Cluster

	Context("SlurmControlInterface()", func() {
		It("Should report control plane is up", func() {
			By("Setup initial system state")
			cluster = &slinkyv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      clusterName,
				},
			}
			controllerPing := &types.V0041ControllerPing{
				V0041ControllerPing: v0041.V0041ControllerPing{
					Hostname: ptr.To("foo"),
					Pinged:   ptr.To(types.V0041ControllerPingPingedUP),
				},
			}
			client := fake.NewClientBuilder().WithObjects(controllerPing).Build()
			clusters := newSlurmClusters(clusterName, client)
			slurmcontrol = NewSlurmControl(clusters)

			By("Pinging the Slurm control plane")
			ok, err := slurmcontrol.PingController(ctx, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("Should report control pane is down", func() {
			By("Setup initial system state")
			cluster = &slinkyv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      clusterName,
				},
			}
			controllerPing := &types.V0041ControllerPing{
				V0041ControllerPing: v0041.V0041ControllerPing{
					Hostname: ptr.To("foo"),
					Pinged:   ptr.To(types.V0041ControllerPingPingedDOWN),
				},
			}
			client := fake.NewClientBuilder().WithObjects(controllerPing).Build()
			clusters := newSlurmClusters(clusterName, client)
			slurmcontrol = NewSlurmControl(clusters)

			By("Pinging the Slurm control plane")
			ok, err := slurmcontrol.PingController(ctx, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})
})

func Test_tolerateError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Nil",
			args: args{
				err: nil,
			},
			want: true,
		},
		{
			name: "Empty",
			args: args{
				err: errors.New(""),
			},
			want: false,
		},
		{
			name: "NotFound",
			args: args{
				err: errors.New(http.StatusText(http.StatusNotFound)),
			},
			want: true,
		},
		{
			name: "NoContent",
			args: args{
				err: errors.New(http.StatusText(http.StatusNoContent)),
			},
			want: true,
		},
		{
			name: "Forbidden",
			args: args{
				err: errors.New(http.StatusText(http.StatusForbidden)),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tolerateError(tt.args.err); got != tt.want {
				t.Errorf("tolerateError() = %v, want %v", got, tt.want)
			}
		})
	}
}

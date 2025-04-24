// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package nodeset

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/controller/history"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	slurminterceptor "github.com/SlinkyProject/slurm-client/pkg/client/interceptor"
	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	nodesetutils "github.com/SlinkyProject/slurm-operator/internal/controller/nodeset/utils"
	"github.com/SlinkyProject/slurm-operator/internal/resources"
	"github.com/SlinkyProject/slurm-operator/internal/utils"
)

func TestNodeSetReconciler_syncStatus(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(scheme.Scheme))
	const clusterName = "slurm"
	const hash = "12345"
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx             context.Context
		nodeset         *slinkyv1alpha1.NodeSet
		pods            []*corev1.Pod
		currentRevision *appsv1.ControllerRevision
		updateRevision  *appsv1.ControllerRevision
		collisionCount  int32
		hash            string
		errors          []error
	}
	type testCaseFields struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	tests := []testCaseFields{
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := make([]*corev1.Pod, 0)
			for i := range 2 {
				pod := nodesetutils.NewNodeSetPod(nodeset, i, hash)
				pod = makePodHealthy(pod)
				pods = append(pods, pod)
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			revision := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						history.ControllerRevisionHashLabel: hash,
					},
				},
			}
			c := fake.NewClientBuilder().WithRuntimeObjects(nodeset, podList, revision).WithStatusSubresource(nodeset).Build()
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: func(pods []*corev1.Pod) []slurmtypes.V0041Node {
					nodeList := make([]slurmtypes.V0041Node, 0, len(pods))
					for _, pod := range pods {
						slurmNode := newNodeSetPodSlurmNode(pod)
						nodeList = append(nodeList, *slurmNode)
					}
					return nodeList
				}(pods),
			}
			sc := newFakeClientList(slurminterceptor.Funcs{}, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, sc)

			return testCaseFields{
				name: "Healthy, up-to-date",
				fields: fields{
					Client:        c,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:             context.TODO(),
					nodeset:         nodeset,
					pods:            pods,
					currentRevision: revision,
					updateRevision:  revision,
					collisionCount:  0,
					hash:            hash,
				},
				wantErr: false,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := make([]*corev1.Pod, 0)
			for i := range 2 {
				pod := nodesetutils.NewNodeSetPod(nodeset, i, hash)
				pod = makePodCreated(pod)
				pods = append(pods, pod)
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			revision := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						history.ControllerRevisionHashLabel: hash,
					},
				},
			}
			c := fake.NewClientBuilder().WithRuntimeObjects(nodeset, podList, revision).WithStatusSubresource(nodeset).Build()
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: func(pods []*corev1.Pod) []slurmtypes.V0041Node {
					nodeList := make([]slurmtypes.V0041Node, 0, len(pods))
					for _, pod := range pods {
						slurmNode := newNodeSetPodSlurmNode(pod)
						nodeList = append(nodeList, *slurmNode)
					}
					return nodeList
				}(pods),
			}
			sc := newFakeClientList(slurminterceptor.Funcs{}, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, sc)

			return testCaseFields{
				name: "Created, need update",
				fields: fields{
					Client:        c,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:             context.TODO(),
					nodeset:         nodeset,
					pods:            pods,
					currentRevision: revision,
					updateRevision:  &appsv1.ControllerRevision{},
					collisionCount:  0,
					hash:            hash,
				},
				wantErr: false,
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.syncStatus(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.currentRevision, tt.args.updateRevision, tt.args.collisionCount, tt.args.hash, tt.args.errors...); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.syncStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_syncSlurmStatus(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(scheme.Scheme))
	const clusterName = "slurm"
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pods    []*corev1.Pod
	}
	type testCaseFields struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	tests := []testCaseFields{
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := make([]*corev1.Pod, 0)
			for i := range 2 {
				pod := nodesetutils.NewNodeSetPod(nodeset, i, "")
				pod = makePodHealthy(pod)
				pods = append(pods, pod)
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			c := fake.NewClientBuilder().WithRuntimeObjects(nodeset, podList).WithStatusSubresource(nodeset).Build()
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: func(pods []*corev1.Pod) []slurmtypes.V0041Node {
					nodeList := make([]slurmtypes.V0041Node, 0, len(pods))
					for _, pod := range pods {
						slurmNode := newNodeSetPodSlurmNode(pod)
						nodeList = append(nodeList, *slurmNode)
					}
					return nodeList
				}(pods),
			}
			sc := newFakeClientList(slurminterceptor.Funcs{}, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, sc)

			return testCaseFields{
				name: "Healthy pods",
				fields: fields{
					Client:        c,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:     context.TODO(),
					nodeset: nodeset,
					pods:    pods,
				},
				wantErr: false,
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.syncSlurmStatus(tt.args.ctx, tt.args.nodeset, tt.args.pods); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.syncSlurmStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_syncNodeSetStatus(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(scheme.Scheme))
	const clusterName = "slurm"
	const hash = "12345"
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx             context.Context
		nodeset         *slinkyv1alpha1.NodeSet
		pods            []*corev1.Pod
		currentRevision *appsv1.ControllerRevision
		updateRevision  *appsv1.ControllerRevision
		collisionCount  int32
		hash            string
	}
	type testCaseFields struct {
		name       string
		fields     fields
		args       args
		wantStatus *slinkyv1alpha1.NodeSetStatus
		wantErr    bool
	}
	tests := []testCaseFields{
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := make([]*corev1.Pod, 0)
			for i := range 2 {
				pod := nodesetutils.NewNodeSetPod(nodeset, i, hash)
				pod = makePodHealthy(pod)
				pods = append(pods, pod)
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			revision := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						history.ControllerRevisionHashLabel: hash,
					},
				},
			}
			c := fake.NewClientBuilder().WithRuntimeObjects(nodeset, podList, revision).WithStatusSubresource(nodeset).Build()
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: func(pods []*corev1.Pod) []slurmtypes.V0041Node {
					nodeList := make([]slurmtypes.V0041Node, 0, len(pods))
					for _, pod := range pods {
						slurmNode := newNodeSetPodSlurmNode(pod)
						nodeList = append(nodeList, *slurmNode)
					}
					return nodeList
				}(pods),
			}
			sc := newFakeClientList(slurminterceptor.Funcs{}, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, sc)

			return testCaseFields{
				name: "Healthy, up-to-date",
				fields: fields{
					Client:        c,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:             context.TODO(),
					nodeset:         nodeset,
					pods:            pods,
					currentRevision: revision,
					updateRevision:  revision,
					collisionCount:  0,
					hash:            hash,
				},
				wantStatus: &slinkyv1alpha1.NodeSetStatus{
					Replicas:          2,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
					UpdatedReplicas:   2,
					SlurmIdle:         2,
					NodeSetHash:       "12345",
					CollisionCount:    ptr.To[int32](0),
					Selector:          "foo=bar",
				},
				wantErr: false,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := make([]*corev1.Pod, 0)
			for i := range 2 {
				pod := nodesetutils.NewNodeSetPod(nodeset, i, hash)
				pod = makePodCreated(pod)
				pods = append(pods, pod)
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			revision := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						history.ControllerRevisionHashLabel: hash,
					},
				},
			}
			c := fake.NewClientBuilder().WithRuntimeObjects(nodeset, podList, revision).WithStatusSubresource(nodeset).Build()
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: func(pods []*corev1.Pod) []slurmtypes.V0041Node {
					nodeList := make([]slurmtypes.V0041Node, 0, len(pods))
					for _, pod := range pods {
						slurmNode := newNodeSetPodSlurmNode(pod)
						nodeList = append(nodeList, *slurmNode)
					}
					return nodeList
				}(pods),
			}
			sc := newFakeClientList(slurminterceptor.Funcs{}, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, sc)

			return testCaseFields{
				name: "Created, need update",
				fields: fields{
					Client:        c,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:             context.TODO(),
					nodeset:         nodeset,
					pods:            pods,
					currentRevision: revision,
					updateRevision:  &appsv1.ControllerRevision{},
					collisionCount:  0,
					hash:            hash,
				},
				wantStatus: &slinkyv1alpha1.NodeSetStatus{
					Replicas:            2,
					UnavailableReplicas: 2,
					NodeSetHash:         "12345",
					CollisionCount:      ptr.To[int32](0),
					Selector:            "foo=bar",
				},
				wantErr: false,
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.syncNodeSetStatus(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.currentRevision, tt.args.updateRevision, tt.args.collisionCount, tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.syncNodeSetStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := &slinkyv1alpha1.NodeSet{}
			key := client.ObjectKeyFromObject(tt.args.nodeset)
			if err := r.Get(tt.args.ctx, key, got); err == nil {
				if diff := cmp.Diff(tt.wantStatus, &got.Status); diff != "" {
					t.Errorf("unexpected status (-want,+got):\n%s", diff)
				}
			}
		})
	}
}

func TestNodeSetReconciler_calculateReplicaStatus(t *testing.T) {
	const clusterName = "slurm"
	const hash = "12345"
	type args struct {
		nodeset         *slinkyv1alpha1.NodeSet
		pods            []*corev1.Pod
		currentRevision *appsv1.ControllerRevision
		updateRevision  *appsv1.ControllerRevision
	}
	tests := []struct {
		name string
		args args
		want replicaStatus
	}{
		{
			name: "Empty",
			args: args{},
			want: replicaStatus{},
		},
		{
			name: "Healthy, up-to-date",
			args: func() args {
				nodeset := newNodeSet("foo", clusterName, 2)
				pods := make([]*corev1.Pod, 0)
				for i := range 2 {
					pod := nodesetutils.NewNodeSetPod(nodeset, i, hash)
					pod = makePodHealthy(pod)
					pods = append(pods, pod)
				}
				revision := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							history.ControllerRevisionHashLabel: hash,
						},
					},
				}
				return args{
					nodeset:         nodeset,
					pods:            pods,
					currentRevision: revision,
					updateRevision:  revision,
				}
			}(),
			want: replicaStatus{
				Replicas:  2,
				Available: 2,
				Ready:     2,
				Current:   2,
				Updated:   2,
			},
		},
		{
			name: "Created, need update",
			args: func() args {
				nodeset := newNodeSet("foo", clusterName, 2)
				pods := make([]*corev1.Pod, 0)
				for i := range 2 {
					pod := nodesetutils.NewNodeSetPod(nodeset, i, hash)
					pod = makePodCreated(pod)
					pods = append(pods, pod)
				}
				revision := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							history.ControllerRevisionHashLabel: hash,
						},
					},
				}
				return args{
					nodeset:         nodeset,
					pods:            pods,
					currentRevision: revision,
					updateRevision:  &appsv1.ControllerRevision{},
				}
			}(),
			want: replicaStatus{
				Replicas:    2,
				Unavailable: 2,
				Current:     2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(fake.NewFakeClient(), nil)
			got := r.calculateReplicaStatus(tt.args.nodeset, tt.args.pods, tt.args.currentRevision, tt.args.updateRevision)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected status (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestNodeSetReconciler_updateNodeSetStatus(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(scheme.Scheme))
	const clusterName = "slurm"
	nodeset := newNodeSet("foo", clusterName, 2)
	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx       context.Context
		nodeset   *slinkyv1alpha1.NodeSet
		newStatus *slinkyv1alpha1.NodeSetStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				Client: fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithRuntimeObjects(nodeset).
					WithStatusSubresource(nodeset).
					Build(),
			},
			args: args{
				ctx:       context.TODO(),
				nodeset:   nodeset,
				newStatus: &slinkyv1alpha1.NodeSetStatus{},
			},
			wantErr: false,
		},
		{
			name: "NotFound",
			fields: fields{
				Client: fake.NewFakeClient(),
			},
			args: args{
				ctx:       context.TODO(),
				nodeset:   nodeset,
				newStatus: &slinkyv1alpha1.NodeSetStatus{},
			},
			wantErr: false,
		},
		{
			name: "Error",
			fields: fields{
				Client: fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithRuntimeObjects(nodeset).
					WithStatusSubresource(nodeset).
					WithInterceptorFuncs(interceptor.Funcs{
						SubResourceUpdate: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
							return errors.New("failed to update resource status")
						},
					}).
					Build(),
			},
			args: args{
				ctx:       context.TODO(),
				nodeset:   nodeset,
				newStatus: &slinkyv1alpha1.NodeSetStatus{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, nil)
			if err := r.updateNodeSetStatus(tt.args.ctx, tt.args.nodeset, tt.args.newStatus); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.updateNodeSetStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := &slinkyv1alpha1.NodeSet{}
			key := client.ObjectKeyFromObject(tt.args.nodeset)
			if err := r.Get(tt.args.ctx, key, got); err == nil {
				if diff := cmp.Diff(tt.args.newStatus, &got.Status); diff != "" {
					t.Errorf("unexpected status (-want,+got):\n%s", diff)
				}
			}
		})
	}
}

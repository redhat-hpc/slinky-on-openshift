// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package nodeset

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	kubecontroller "k8s.io/kubernetes/pkg/controller"
	"k8s.io/kubernetes/pkg/controller/history"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	v0041 "github.com/SlinkyProject/slurm-client/api/v0041"
	slurmclient "github.com/SlinkyProject/slurm-client/pkg/client"
	sinterceptor "github.com/SlinkyProject/slurm-client/pkg/client/interceptor"
	slurmobject "github.com/SlinkyProject/slurm-client/pkg/object"
	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	"github.com/SlinkyProject/slurm-operator/internal/controller/nodeset/podcontrol"
	"github.com/SlinkyProject/slurm-operator/internal/controller/nodeset/slurmcontrol"
	nodesetutils "github.com/SlinkyProject/slurm-operator/internal/controller/nodeset/utils"
	"github.com/SlinkyProject/slurm-operator/internal/resources"
	"github.com/SlinkyProject/slurm-operator/internal/utils"
	"github.com/SlinkyProject/slurm-operator/internal/utils/historycontrol"
)

func newNodeSetController(client client.Client, slurmClusters *resources.Clusters) *NodeSetReconciler {
	eventRecorder := record.NewFakeRecorder(10)
	r := &NodeSetReconciler{
		Client:         client,
		Scheme:         client.Scheme(),
		SlurmClusters:  slurmClusters,
		eventRecorder:  eventRecorder,
		historyControl: historycontrol.NewHistoryControl(client),
		podControl:     podcontrol.NewPodControl(client, eventRecorder),
		slurmControl:   slurmcontrol.NewSlurmControl(slurmClusters),
		expectations:   kubecontroller.NewUIDTrackingControllerExpectations(kubecontroller.NewControllerExpectations()),
	}
	return r
}

func newNodeSet(name, clusterName string, replicas int32) *slinkyv1alpha1.NodeSet {
	return &slinkyv1alpha1.NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: corev1.NamespaceDefault,
			Name:      name,
		},
		Spec: slinkyv1alpha1.NodeSetSpec{
			ClusterName: clusterName,
			Selector: metav1.SetAsLabelSelector(labels.Set{
				"foo": "bar",
			}),
			Replicas: ptr.To(replicas),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
		},
	}
}

func newSlurmClusters(clusterName string, client slurmclient.Client) *resources.Clusters {
	clusters := resources.NewClusters()
	key := types.NamespacedName{
		Namespace: corev1.NamespaceDefault,
		Name:      clusterName,
	}
	clusters.Add(key, client)
	return clusters
}

func newNodeSetPodSlurmNode(pod *corev1.Pod) *slurmtypes.V0041Node {
	node := &slurmtypes.V0041Node{
		V0041Node: v0041.V0041Node{
			Name: ptr.To(pod.GetName()),
		},
	}
	switch {
	case utils.IsPending(pod):
		node.State = nil
	default:
		node.State = ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE})
	}
	return node
}

func makePodCreated(pod *corev1.Pod) *corev1.Pod {
	pod.Status.Phase = corev1.PodPending
	return pod
}

func makePodHealthy(pod *corev1.Pod) *corev1.Pod {
	pod.Status.Phase = corev1.PodRunning
	podCond := corev1.PodCondition{
		Type:   corev1.PodReady,
		Status: corev1.ConditionTrue,
	}
	pod.Status.Conditions = append(pod.Status.Conditions, podCond)
	return pod
}

func TestNodeSetReconciler_adoptOrphanRevisions(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "No revisions",
			fields: fields{
				Client: fake.NewFakeClient(),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: newNodeSet("foo", clusterName, 2),
			},
			wantErr: false,
		},
		{
			name: "Adopt the revision",
			fields: fields{
				Client: fake.NewFakeClient(newNodeSet("foo", clusterName, 2), &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "foo-00000",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				}),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: newNodeSet("foo", clusterName, 2),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, nil)
			if err := r.adoptOrphanRevisions(tt.args.ctx, tt.args.nodeset); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.adoptOrphanRevisions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_doAdoptOrphanRevisions(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	type fields struct {
		Client client.Client
	}
	type args struct {
		nodeset   *slinkyv1alpha1.NodeSet
		revisions []*appsv1.ControllerRevision
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "No revisions",
			fields: fields{
				Client: fake.NewFakeClient(),
			},
			args: args{
				nodeset:   newNodeSet("foo", clusterName, 2),
				revisions: []*appsv1.ControllerRevision{},
			},
			wantErr: false,
		},
		{
			name: "Adopt revision",
			fields: fields{
				Client: fake.NewFakeClient(&appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "foo-00000",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				}),
			},
			args: args{
				nodeset: newNodeSet("foo", clusterName, 2),
				revisions: []*appsv1.ControllerRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: corev1.NamespaceDefault,
							Name:      "foo-00000",
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, nil)
			if err := r.doAdoptOrphanRevisions(tt.args.nodeset, tt.args.revisions); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.doAdoptOrphanRevisions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_listRevisions(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		nodeset *slinkyv1alpha1.NodeSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*appsv1.ControllerRevision
		wantErr bool
	}{
		{
			name: "Empty",
			fields: fields{
				Client: fake.NewFakeClient(),
			},
			args: args{
				nodeset: newNodeSet("foo", clusterName, 2),
			},
			want:    []*appsv1.ControllerRevision{},
			wantErr: false,
		},
		{
			name: "Has revisions",
			fields: fields{
				Client: fake.NewFakeClient(&appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "foo-00000",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				}),
			},
			args: args{
				nodeset: newNodeSet("foo", clusterName, 2),
			},
			want: []*appsv1.ControllerRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "foo-00000",
						Labels: map[string]string{
							"foo": "bar",
						},
						ResourceVersion: "999",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "No matching labels",
			fields: fields{
				Client: fake.NewFakeClient(&appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "foo-00000",
					},
				}),
			},
			args: args{
				nodeset: newNodeSet("foo", clusterName, 2),
			},
			want:    []*appsv1.ControllerRevision{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, nil)
			got, err := r.listRevisions(tt.args.nodeset)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.listRevisions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !apiequality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("NodeSetReconciler.listRevisions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeSetReconciler_getNodeSetPods(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	nodeset := newNodeSet("foo", clusterName, 2)
	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "selector match",
			fields: fields{
				Client: fake.NewFakeClient(
					nodeset.DeepCopy(),
					nodesetutils.NewNodeSetPod(nodeset, 0, ""),
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "blank",
						},
					}),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: nodeset.DeepCopy(),
			},
			want:    []string{klog.KObj(nodesetutils.NewNodeSetPod(nodeset, 0, "")).String()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, nil)
			got, err := r.getNodeSetPods(tt.args.ctx, tt.args.nodeset)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.getNodeSetPods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotPodNames := make([]string, len(got))
			for i, pod := range got {
				gotPodNames[i] = klog.KObj(pod).String()
			}
			if diff := cmp.Diff(tt.want, gotPodNames); diff != "" {
				t.Errorf("NodeSetReconciler.getNodeSetPods() (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestNodeSetReconciler_sync(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pods    []*corev1.Pod
		hash    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.sync(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_syncNodeSet(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pods    []*corev1.Pod
		hash    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.syncNodeSet(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.syncNodeSet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_doPodScaleOut(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx       context.Context
		nodeset   *slinkyv1alpha1.NodeSet
		pods      []*corev1.Pod
		numCreate int
		hash      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.doPodScaleOut(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.numCreate, tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.doPodScaleOut() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_doPodScaleIn(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx          context.Context
		nodeset      *slinkyv1alpha1.NodeSet
		podsToDelete []*corev1.Pod
		podsToKeep   []*corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.doPodScaleIn(tt.args.ctx, tt.args.nodeset, tt.args.podsToDelete, tt.args.podsToKeep); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.doPodScaleIn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_processCondemned(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx       context.Context
		nodeset   *slinkyv1alpha1.NodeSet
		condemned []*corev1.Pod
		i         int
	}
	type testCaseFields struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		wantDrain  bool
		wantDelete bool
	}
	tests := []testCaseFields{
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "pod-0",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			client := fake.NewFakeClient(nodeset, podList)
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pods[0])),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
				},
			}
			slurmClient := newFakeClientList(sinterceptor.Funcs{}, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, slurmClient)

			return testCaseFields{
				name: "drain",
				fields: fields{
					Client:        client,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:       context.TODO(),
					nodeset:   nodeset,
					condemned: pods,
					i:         0,
				},
				wantErr:    false,
				wantDrain:  true,
				wantDelete: false,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "pod-0",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			client := fake.NewFakeClient(nodeset, podList)
			slurmClient := newFakeClientList(sinterceptor.Funcs{})
			slurmClusters := newSlurmClusters(clusterName, slurmClient)

			return testCaseFields{
				name: "delete",
				fields: fields{
					Client:        client,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:       context.TODO(),
					nodeset:   nodeset,
					condemned: pods,
					i:         0,
				},
				wantErr:    false,
				wantDrain:  false,
				wantDelete: true,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "pod-0",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			client := fake.NewFakeClient(nodeset, podList)
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name: ptr.To(nodesetutils.GetNodeName(pods[0])),
							State: ptr.To([]v0041.V0041NodeState{
								v0041.V0041NodeStateIDLE,
								v0041.V0041NodeStateDRAIN,
							}),
						},
					},
				},
			}
			slurmClient := newFakeClientList(sinterceptor.Funcs{}, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, slurmClient)

			return testCaseFields{
				name: "delete after drain",
				fields: fields{
					Client:        client,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:       context.TODO(),
					nodeset:   nodeset,
					condemned: pods,
					i:         0,
				},
				wantErr:    false,
				wantDrain:  true,
				wantDelete: false,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "pod-0",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			client := fake.NewClientBuilder().
				WithInterceptorFuncs(interceptor.Funcs{
					Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
						return http.ErrHandlerTimeout
					},
					Patch: func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						return http.ErrHandlerTimeout
					},
				}).
				WithRuntimeObjects(nodeset, podList).
				Build()
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pods[0])),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
				},
			}
			slurmClient := newFakeClientList(sinterceptor.Funcs{}, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, slurmClient)

			return testCaseFields{
				name: "k8s error",
				fields: fields{
					Client:        client,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:       context.TODO(),
					nodeset:   nodeset,
					condemned: pods,
					i:         0,
				},
				wantErr:    true,
				wantDrain:  false,
				wantDelete: false,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			pods := []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "pod-0",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			}
			podList := &corev1.PodList{
				Items: utils.DereferenceList(pods),
			}
			client := fake.NewFakeClient(nodeset, podList)
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name: ptr.To(nodesetutils.GetNodeName(pods[0])),
						},
					},
				},
			}
			slurmInterceptorFn := sinterceptor.Funcs{
				Update: func(ctx context.Context, obj slurmobject.Object, req any, opts ...slurmclient.UpdateOption) error {
					return http.ErrHandlerTimeout
				},
			}
			slurmClient := newFakeClientList(slurmInterceptorFn, slurmNodeList)
			slurmClusters := newSlurmClusters(clusterName, slurmClient)

			return testCaseFields{
				name: "slurm error",
				fields: fields{
					Client:        client,
					SlurmClusters: slurmClusters,
				},
				args: args{
					ctx:       context.TODO(),
					nodeset:   nodeset,
					condemned: pods,
					i:         0,
				},
				wantErr:    true,
				wantDrain:  false,
				wantDelete: false,
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.processCondemned(tt.args.ctx, tt.args.nodeset, tt.args.condemned, tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.processCondemned() error = %v, wantErr %v", err, tt.wantErr)
			}
			pod := tt.args.condemned[tt.args.i]
			if isDrain, err := r.slurmControl.IsNodeDrain(tt.args.ctx, tt.args.nodeset, pod); err != nil {
				t.Errorf("slurmControl.IsNodeDrain() error = %v", err)
			} else if isDrain != tt.wantDrain && !tt.wantDelete {
				t.Errorf("slurmControl.IsNodeDrain() = %v, wantDrain %v", isDrain, tt.wantDrain)
			}
			key := client.ObjectKeyFromObject(pod)
			if err := r.Get(tt.args.ctx, key, pod); err != nil && !apierrors.IsNotFound(err) {
				t.Errorf("Client.Get() error = %v, wantDelete %v", err, tt.wantDelete)
			}
		})
	}
}

func TestNodeSetReconciler_doPodProcessing(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pods    []*corev1.Pod
		hash    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.doPodProcessing(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.doPodProcessing() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_processReplica(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pod     *corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.processReplica(tt.args.ctx, tt.args.nodeset, tt.args.pod); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.processReplica() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_makePodCordonAndDrain(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	nodeset := newNodeSet("foo", clusterName, 2)
	pod := nodesetutils.NewNodeSetPod(nodeset, 0, "")
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pod     *corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Client: fake.NewFakeClient(nodeset.DeepCopy(), pod.DeepCopy()),
				SlurmClusters: func() *resources.Clusters {
					nodeList := &slurmtypes.V0041NodeList{
						Items: []slurmtypes.V0041Node{
							{
								V0041Node: v0041.V0041Node{
									Name:  ptr.To(nodesetutils.GetNodeName(pod)),
									State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
								},
							},
						},
					}
					sclient := newFakeClientList(sinterceptor.Funcs{}, nodeList)
					return newSlurmClusters(clusterName, sclient)
				}(),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: nodeset.DeepCopy(),
				pod:     pod.DeepCopy(),
			},
			wantErr: false,
		},
		{
			name: "kubernetes update failure",
			fields: fields{
				Client: fake.NewClientBuilder().
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							return http.ErrAbortHandler
						},
						Patch: func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
							return http.ErrAbortHandler
						},
					}).
					WithRuntimeObjects(nodeset.DeepCopy(), pod.DeepCopy()).
					Build(),
				SlurmClusters: func() *resources.Clusters {
					nodeList := &slurmtypes.V0041NodeList{
						Items: []slurmtypes.V0041Node{
							{
								V0041Node: v0041.V0041Node{
									Name:  ptr.To(nodesetutils.GetNodeName(pod)),
									State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
								},
							},
						},
					}
					sclient := newFakeClientList(sinterceptor.Funcs{}, nodeList)
					return newSlurmClusters(clusterName, sclient)
				}(),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: nodeset.DeepCopy(),
				pod:     pod.DeepCopy(),
			},
			wantErr: true,
		},
		{
			name: "slurm update failure",
			fields: fields{
				Client: fake.NewFakeClient(nodeset.DeepCopy(), pod.DeepCopy()),
				SlurmClusters: func() *resources.Clusters {
					nodeList := &slurmtypes.V0041NodeList{
						Items: []slurmtypes.V0041Node{
							{
								V0041Node: v0041.V0041Node{
									Name:  ptr.To(nodesetutils.GetNodeName(pod)),
									State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
								},
							},
						},
					}
					sclient := newFakeClientList(sinterceptor.Funcs{
						Update: func(ctx context.Context, obj slurmobject.Object, req any, opts ...slurmclient.UpdateOption) error {
							return errors.New(http.StatusText(http.StatusInternalServerError))
						},
					}, nodeList)
					return newSlurmClusters(clusterName, sclient)
				}(),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: nodeset.DeepCopy(),
				pod:     pod.DeepCopy(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.makePodCordonAndDrain(tt.args.ctx, tt.args.nodeset, tt.args.pod); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.makePodCordonAndDrain() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Check Pod Annotations
			gotPod := &corev1.Pod{}
			if err := r.Get(tt.args.ctx, client.ObjectKeyFromObject(tt.args.pod), gotPod); err != nil {
				if !apierrors.IsNotFound(err) {
					t.Errorf("client.Get() error = %v", err)
				}
			} else if !tt.wantErr {
				if ok := utils.IsPodCordon(gotPod); !ok {
					t.Errorf("IsPodCordon() = %v", ok)
				}
			}
			// Check Slurm Node State
			gotSlurmNode := &slurmtypes.V0041Node{}
			sc := r.SlurmClusters.Get(types.NamespacedName{Namespace: tt.args.nodeset.GetNamespace(), Name: tt.args.nodeset.Spec.ClusterName})
			if sc == nil {
				t.Error("SlurmClusters.Get() is nil")
			}
			if err := sc.Get(tt.args.ctx, slurmclient.ObjectKey(nodesetutils.GetNodeName(tt.args.pod)), gotSlurmNode); err != nil {
				if err.Error() != http.StatusText(http.StatusNotFound) {
					t.Errorf("slurmclient.Get() error = %v", err)
				}
			} else if !tt.wantErr {
				if ok := gotSlurmNode.GetStateAsSet().Has(v0041.V0041NodeStateDRAIN); !ok {
					t.Errorf("SlurmNode Has DRAIN = %v", ok)
				}
			}
		})
	}
}

func TestNodeSetReconciler_makePodCordon(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-0",
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-1",
			Annotations: map[string]string{
				slinkyv1alpha1.AnnotationPodCordon: "true",
			},
		},
	}
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx context.Context
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{

		{
			name: "NotFound",
			fields: fields{
				Client: fake.NewFakeClient(),
			},
			args: args{
				ctx: context.TODO(),
				pod: pod1.DeepCopy(),
			},
			wantErr: true,
		},
		{
			name: "cordoned",
			fields: fields{
				Client: fake.NewFakeClient(pod2.DeepCopy()),
			},
			args: args{
				ctx: context.TODO(),
				pod: pod2.DeepCopy(),
			},
			wantErr: false,
		},
		{
			name: "not cordoned",
			fields: fields{
				Client: fake.NewFakeClient(pod1.DeepCopy()),
			},
			args: args{
				ctx: context.TODO(),
				pod: pod1.DeepCopy(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.makePodCordon(tt.args.ctx, tt.args.pod); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.makePodCordon() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Check Pod Annotations
			gotPod := &corev1.Pod{}
			if err := r.Get(tt.args.ctx, client.ObjectKeyFromObject(tt.args.pod), gotPod); err != nil {
				if !apierrors.IsNotFound(err) {
					t.Errorf("client.Get() error = %v", err)
				}
			} else if !tt.wantErr {
				if ok := utils.IsPodCordon(gotPod); !ok {
					t.Errorf("IsPodCordon() = %v", ok)
				}
			}
		})
	}
}

func TestNodeSetReconciler_makePodUncordonAndUndrain(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	nodeset := newNodeSet("foo", clusterName, 2)
	pod := nodesetutils.NewNodeSetPod(nodeset, 0, "")
	pod.Annotations[slinkyv1alpha1.AnnotationPodCordon] = "true"
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pod     *corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Client: fake.NewFakeClient(nodeset.DeepCopy(), pod.DeepCopy()),
				SlurmClusters: func() *resources.Clusters {
					nodeList := &slurmtypes.V0041NodeList{
						Items: []slurmtypes.V0041Node{
							{
								V0041Node: v0041.V0041Node{
									Name: ptr.To(nodesetutils.GetNodeName(pod)),
									State: ptr.To([]v0041.V0041NodeState{
										v0041.V0041NodeStateIDLE,
										v0041.V0041NodeStateDRAIN,
									}),
								},
							},
						},
					}
					sclient := newFakeClientList(sinterceptor.Funcs{}, nodeList)
					return newSlurmClusters(clusterName, sclient)
				}(),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: nodeset.DeepCopy(),
				pod:     pod.DeepCopy(),
			},
			wantErr: false,
		},
		{
			name: "kubernetes update failure",
			fields: fields{
				Client: fake.NewClientBuilder().
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							return http.ErrAbortHandler
						},
						Patch: func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
							return http.ErrAbortHandler
						},
					}).
					WithRuntimeObjects(nodeset.DeepCopy(), pod.DeepCopy()).
					Build(),
				SlurmClusters: func() *resources.Clusters {
					nodeList := &slurmtypes.V0041NodeList{
						Items: []slurmtypes.V0041Node{
							{
								V0041Node: v0041.V0041Node{
									Name: ptr.To(nodesetutils.GetNodeName(pod)),
									State: ptr.To([]v0041.V0041NodeState{
										v0041.V0041NodeStateIDLE,
										v0041.V0041NodeStateDRAIN,
									}),
								},
							},
						},
					}
					sclient := newFakeClientList(sinterceptor.Funcs{}, nodeList)
					return newSlurmClusters(clusterName, sclient)
				}(),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: nodeset.DeepCopy(),
				pod:     pod.DeepCopy(),
			},
			wantErr: true,
		},
		{
			name: "slurm update failure",
			fields: fields{
				Client: fake.NewFakeClient(nodeset.DeepCopy(), pod.DeepCopy()),
				SlurmClusters: func() *resources.Clusters {
					nodeList := &slurmtypes.V0041NodeList{
						Items: []slurmtypes.V0041Node{
							{
								V0041Node: v0041.V0041Node{
									Name: ptr.To(nodesetutils.GetNodeName(pod)),
									State: ptr.To([]v0041.V0041NodeState{
										v0041.V0041NodeStateIDLE,
										v0041.V0041NodeStateDRAIN,
									}),
								},
							},
						},
					}
					sclient := newFakeClientList(sinterceptor.Funcs{
						Update: func(ctx context.Context, obj slurmobject.Object, req any, opts ...slurmclient.UpdateOption) error {
							return errors.New(http.StatusText(http.StatusInternalServerError))
						},
					}, nodeList)
					return newSlurmClusters(clusterName, sclient)
				}(),
			},
			args: args{
				ctx:     context.TODO(),
				nodeset: nodeset.DeepCopy(),
				pod:     pod.DeepCopy(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.makePodUncordonAndUndrain(tt.args.ctx, tt.args.nodeset, tt.args.pod); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.makePodUncordonAndUndrain() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Check Pod Annotations
			gotPod := &corev1.Pod{}
			if err := r.Get(tt.args.ctx, client.ObjectKeyFromObject(tt.args.pod), gotPod); err != nil {
				if !apierrors.IsNotFound(err) {
					t.Errorf("client.Get() error = %v", err)
				}
			} else if !tt.wantErr {
				if ok := utils.IsPodCordon(gotPod); ok {
					t.Errorf("IsPodCordon() = %v", ok)
				}
			}
			// Check Slurm Node State
			gotSlurmNode := &slurmtypes.V0041Node{}
			sc := r.SlurmClusters.Get(types.NamespacedName{Namespace: tt.args.nodeset.GetNamespace(), Name: tt.args.nodeset.Spec.ClusterName})
			if sc == nil {
				t.Error("SlurmClusters.Get() is nil")
			}
			if err := sc.Get(tt.args.ctx, slurmclient.ObjectKey(nodesetutils.GetNodeName(tt.args.pod)), gotSlurmNode); err != nil {
				if err.Error() != http.StatusText(http.StatusNotFound) {
					t.Errorf("slurmclient.Get() error = %v", err)
				}
			} else if !tt.wantErr {
				if ok := gotSlurmNode.GetStateAsSet().Has(v0041.V0041NodeStateDRAIN); ok {
					t.Errorf("SlurmNode Has DRAIN = %v", ok)
				}
			}
		})
	}
}

func TestNodeSetReconciler_makePodUncordon(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-0",
			Annotations: map[string]string{
				slinkyv1alpha1.AnnotationPodCordon: "true",
			},
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-1",
		},
	}
	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx context.Context
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "NotFound",
			fields: fields{
				Client: fake.NewFakeClient(),
			},
			args: args{
				ctx: context.TODO(),
				pod: pod1.DeepCopy(),
			},
			wantErr: true,
		},
		{
			name: "cordoned",
			fields: fields{
				Client: fake.NewFakeClient(pod1.DeepCopy()),
			},
			args: args{
				ctx: context.TODO(),
				pod: pod1.DeepCopy(),
			},
			wantErr: false,
		},
		{
			name: "not cordoned",
			fields: fields{
				Client: fake.NewFakeClient(pod2.DeepCopy()),
			},
			args: args{
				ctx: context.TODO(),
				pod: pod2.DeepCopy(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, nil)
			if err := r.makePodUncordon(tt.args.ctx, tt.args.pod); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.makePodUncordon() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Check Pod Annotations
			gotPod := &corev1.Pod{}
			if err := r.Get(tt.args.ctx, client.ObjectKeyFromObject(tt.args.pod), gotPod); err != nil {
				if !apierrors.IsNotFound(err) {
					t.Errorf("client.Get() error = %v", err)
				}
			} else if !tt.wantErr {
				if ok := utils.IsPodCordon(gotPod); ok {
					t.Errorf("IsPodCordon() = %v", ok)
				}
			}
		})
	}
}

func TestNodeSetReconciler_syncUpdate(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	const hash = "12345"
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pods    []*corev1.Pod
		hash    string
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
			nodeset.Spec.UpdateStrategy.Type = slinkyv1alpha1.OnDeleteNodeSetStrategyType
			pod1 := nodesetutils.NewNodeSetPod(nodeset, 0, hash)
			pod2 := nodesetutils.NewNodeSetPod(nodeset, 1, "")
			k8sclient := fake.NewFakeClient(nodeset, pod1, pod2)
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod1)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod2)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
				},
			}
			slurmClient := newFakeClientList(sinterceptor.Funcs{}, slurmNodeList)
			return testCaseFields{
				name: "OnDelete",
				fields: fields{
					Client:        k8sclient,
					SlurmClusters: newSlurmClusters(clusterName, slurmClient),
				},
				args: args{
					ctx:     context.TODO(),
					nodeset: nodeset,
					pods:    []*corev1.Pod{pod1, pod2},
					hash:    hash,
				},
				wantErr: false,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			nodeset.Spec.UpdateStrategy.Type = slinkyv1alpha1.RollingUpdateNodeSetStrategyType
			nodeset.Spec.UpdateStrategy.RollingUpdate = &slinkyv1alpha1.RollingUpdateNodeSetStrategy{
				MaxUnavailable: ptr.To(intstr.FromString("10%")),
			}
			pod1 := nodesetutils.NewNodeSetPod(nodeset, 0, hash)
			pod2 := nodesetutils.NewNodeSetPod(nodeset, 1, "")
			k8sclient := fake.NewFakeClient(nodeset, pod1, pod2)
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod1)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod2)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
				},
			}
			slurmClient := newFakeClientList(sinterceptor.Funcs{}, slurmNodeList)
			return testCaseFields{
				name: "RollingUpdate",
				fields: fields{
					Client:        k8sclient,
					SlurmClusters: newSlurmClusters(clusterName, slurmClient),
				},
				args: args{
					ctx:     context.TODO(),
					nodeset: nodeset,
					pods:    []*corev1.Pod{pod1, pod2},
					hash:    hash,
				},
				wantErr: false,
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.syncUpdate(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.syncUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_syncRollingUpdate(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	const hash = "12345"
	type fields struct {
		Client        client.Client
		SlurmClusters *resources.Clusters
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pods    []*corev1.Pod
		hash    string
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
			nodeset.Spec.UpdateStrategy.Type = slinkyv1alpha1.RollingUpdateNodeSetStrategyType
			nodeset.Spec.UpdateStrategy.RollingUpdate = &slinkyv1alpha1.RollingUpdateNodeSetStrategy{
				MaxUnavailable: ptr.To(intstr.FromString("10%")),
			}
			pod1 := nodesetutils.NewNodeSetPod(nodeset, 0, hash)
			makePodHealthy(pod1)
			pod2 := nodesetutils.NewNodeSetPod(nodeset, 1, "")
			makePodHealthy(pod2)
			k8sclient := fake.NewFakeClient(nodeset, pod1, pod2)
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod1)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod2)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
				},
			}
			slurmClient := newFakeClientList(sinterceptor.Funcs{}, slurmNodeList)
			return testCaseFields{
				name: "update",
				fields: fields{
					Client:        k8sclient,
					SlurmClusters: newSlurmClusters(clusterName, slurmClient),
				},
				args: args{
					ctx:     context.TODO(),
					nodeset: nodeset,
					pods:    []*corev1.Pod{pod1, pod2},
					hash:    hash,
				},
				wantErr: false,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			nodeset.Spec.UpdateStrategy.Type = slinkyv1alpha1.RollingUpdateNodeSetStrategyType
			nodeset.Spec.UpdateStrategy.RollingUpdate = &slinkyv1alpha1.RollingUpdateNodeSetStrategy{
				MaxUnavailable: ptr.To(intstr.FromString("10%")),
			}
			pod1 := nodesetutils.NewNodeSetPod(nodeset, 0, hash)
			makePodHealthy(pod1)
			pod2 := nodesetutils.NewNodeSetPod(nodeset, 1, hash)
			makePodHealthy(pod2)
			k8sclient := fake.NewFakeClient(nodeset, pod1, pod2)
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod1)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod2)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
				},
			}
			slurmClient := newFakeClientList(sinterceptor.Funcs{}, slurmNodeList)
			return testCaseFields{
				name: "no update",
				fields: fields{
					Client:        k8sclient,
					SlurmClusters: newSlurmClusters(clusterName, slurmClient),
				},
				args: args{
					ctx:     context.TODO(),
					nodeset: nodeset,
					pods:    []*corev1.Pod{pod1, pod2},
					hash:    hash,
				},
				wantErr: false,
			}
		}(),
		func() testCaseFields {
			nodeset := newNodeSet("foo", clusterName, 2)
			nodeset.Spec.UpdateStrategy.Type = slinkyv1alpha1.RollingUpdateNodeSetStrategyType
			nodeset.Spec.UpdateStrategy.RollingUpdate = &slinkyv1alpha1.RollingUpdateNodeSetStrategy{
				MaxUnavailable: ptr.To(intstr.FromString("10%")),
			}
			pod1 := nodesetutils.NewNodeSetPod(nodeset, 0, "")
			makePodHealthy(pod1)
			pod2 := nodesetutils.NewNodeSetPod(nodeset, 1, "")
			k8sclient := fake.NewFakeClient(nodeset, pod1, pod2)
			slurmNodeList := &slurmtypes.V0041NodeList{
				Items: []slurmtypes.V0041Node{
					{
						V0041Node: v0041.V0041Node{
							Name:  ptr.To(nodesetutils.GetNodeName(pod1)),
							State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
						},
					},
				},
			}
			slurmClient := newFakeClientList(sinterceptor.Funcs{}, slurmNodeList)
			return testCaseFields{
				name: "update, with unhealthy",
				fields: fields{
					Client:        k8sclient,
					SlurmClusters: newSlurmClusters(clusterName, slurmClient),
				},
				args: args{
					ctx:     context.TODO(),
					nodeset: nodeset,
					pods:    []*corev1.Pod{pod1, pod2},
					hash:    hash,
				},
				wantErr: false,
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, tt.fields.SlurmClusters)
			if err := r.syncRollingUpdate(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("NodeSetReconciler.syncRollingUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeSetReconciler_splitUpdatePods(t *testing.T) {
	utilruntime.Must(slinkyv1alpha1.AddToScheme(clientgoscheme.Scheme))
	const clusterName = "slurm"
	now := metav1.Now()
	const hash = "12345"
	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx     context.Context
		nodeset *slinkyv1alpha1.NodeSet
		pods    []*corev1.Pod
		hash    string
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantPodsToDelete []string
		wantPodsToKeep   []string
	}{
		{
			name: "OnDelete",
			fields: fields{
				Client: fake.NewFakeClient(),
			},
			args: args{
				ctx: context.TODO(),
				nodeset: func() *slinkyv1alpha1.NodeSet {
					nodeset := newNodeSet("foo", clusterName, 0)
					nodeset.Spec.UpdateStrategy.Type = slinkyv1alpha1.OnDeleteNodeSetStrategyType
					return nodeset
				}(),
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-0",
							Labels: map[string]string{
								history.ControllerRevisionHashLabel: hash,
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodFailed,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
							Labels: map[string]string{
								history.ControllerRevisionHashLabel: "",
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:               corev1.PodReady,
									Status:             corev1.ConditionTrue,
									LastTransitionTime: now,
								},
							},
						},
					},
				},
				hash: hash,
			},
			wantPodsToDelete: []string{},
			wantPodsToKeep:   []string{},
		},
		{
			name: "RollingUpdate",
			fields: fields{
				Client: fake.NewFakeClient(),
			},
			args: args{
				ctx: context.TODO(),
				nodeset: func() *slinkyv1alpha1.NodeSet {
					nodeset := newNodeSet("foo", clusterName, 0)
					nodeset.Spec.UpdateStrategy.Type = slinkyv1alpha1.RollingUpdateNodeSetStrategyType
					nodeset.Spec.UpdateStrategy.RollingUpdate = &slinkyv1alpha1.RollingUpdateNodeSetStrategy{
						MaxUnavailable: ptr.To(intstr.FromString("100%")),
					}
					return nodeset
				}(),
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-0",
							Labels: map[string]string{
								history.ControllerRevisionHashLabel: hash,
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodFailed,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
							Labels: map[string]string{
								history.ControllerRevisionHashLabel: "",
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:               corev1.PodReady,
									Status:             corev1.ConditionTrue,
									LastTransitionTime: now,
								},
							},
						},
					},
				},
				hash: hash,
			},
			wantPodsToDelete: []string{},
			wantPodsToKeep:   []string{"pod-0", "pod-1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNodeSetController(tt.fields.Client, nil)
			gotPodsToDelete, gotPodsToKeep := r.splitUpdatePods(tt.args.ctx, tt.args.nodeset, tt.args.pods, tt.args.hash)

			gotPodsToDeleteOrdered := make([]string, len(gotPodsToDelete))
			for i := range gotPodsToDelete {
				gotPodsToDeleteOrdered[i] = gotPodsToDelete[i].Name
			}
			gotPodsToKeepOrdered := make([]string, len(gotPodsToKeep))
			for i := range gotPodsToKeep {
				gotPodsToKeepOrdered[i] = gotPodsToKeep[i].Name
			}

			slices.Sort(gotPodsToDeleteOrdered)
			slices.Sort(gotPodsToKeepOrdered)
			if diff := cmp.Diff(tt.wantPodsToDelete, gotPodsToDeleteOrdered); diff != "" {
				t.Errorf("gotPodsToDelete (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantPodsToKeep, gotPodsToKeepOrdered); diff != "" {
				t.Errorf("gotPodsToKeep (-want,+got):\n%s", diff)
			}
		})
	}
}

func Test_findUpdatedPods(t *testing.T) {
	type args struct {
		pods []*corev1.Pod
		hash string
	}
	tests := []struct {
		name        string
		args        args
		wantNewPods []string
		wantOldPods []string
	}{
		{
			name: "1 new, 1 old",
			args: func() args {
				const hash = "12345"
				pods := []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-0",
							Labels: map[string]string{
								history.ControllerRevisionHashLabel: hash,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
							Labels: map[string]string{
								history.ControllerRevisionHashLabel: "",
							},
						},
					},
				}
				return args{
					pods: pods,
					hash: hash,
				}
			}(),
			wantNewPods: []string{"pod-0"},
			wantOldPods: []string{"pod-1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewPods, gotOldPods := findUpdatedPods(tt.args.pods, tt.args.hash)

			gotNewPodsOrdered := make([]string, len(gotNewPods))
			for i := range tt.wantNewPods {
				gotNewPodsOrdered[i] = gotNewPods[i].Name
			}
			gotOldPodsOrdered := make([]string, len(gotOldPods))
			for i := range tt.wantNewPods {
				gotOldPodsOrdered[i] = gotOldPods[i].Name
			}

			slices.Sort(gotNewPodsOrdered)
			slices.Sort(gotOldPodsOrdered)
			if diff := cmp.Diff(tt.wantNewPods, gotNewPodsOrdered); diff != "" {
				t.Errorf("gotNewPods (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantOldPods, gotOldPodsOrdered); diff != "" {
				t.Errorf("gotOldPods (-want,+got):\n%s", diff)
			}
		})
	}
}

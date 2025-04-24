// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package nodeset

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v0041 "github.com/SlinkyProject/slurm-client/api/v0041"
	slurmclient "github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
	"github.com/SlinkyProject/slurm-client/pkg/client/interceptor"
	"github.com/SlinkyProject/slurm-client/pkg/object"
	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	nodesetutils "github.com/SlinkyProject/slurm-operator/internal/controller/nodeset/utils"
)

func newFakeClientList(interceptorFuncs interceptor.Funcs, initObjLists ...object.ObjectList) slurmclient.Client {
	updateFn := func(_ context.Context, obj object.Object, req any, opts ...slurmclient.UpdateOption) error {
		switch o := obj.(type) {
		case *slurmtypes.V0041Node:
			r, ok := req.(v0041.V0041UpdateNodeMsg)
			if !ok {
				return errors.New("failed to cast request object")
			}
			stateSet := set.New(ptr.Deref(o.State, []v0041.V0041NodeState{})...)
			statesReq := ptr.Deref(r.State, []v0041.V0041UpdateNodeMsgState{})
			for _, stateReq := range statesReq {
				switch stateReq {
				case v0041.V0041UpdateNodeMsgStateUNDRAIN:
					stateSet.Delete(v0041.V0041NodeStateDRAIN)
				default:
					stateSet.Insert(v0041.V0041NodeState(stateReq))
				}
			}
			o.State = ptr.To(stateSet.UnsortedList())
			o.Comment = r.Comment
			o.Reason = r.Reason
		default:
			return errors.New("failed to cast slurm object")
		}
		return nil
	}

	return fake.NewClientBuilder().
		WithUpdateFn(updateFn).
		WithLists(initObjLists...).
		WithInterceptorFuncs(interceptorFuncs).
		Build()
}

var _ = Describe("Nodeset controller", func() {

	const (
		nodesetName      = "test-nodeset"
		nodesetNamespace = corev1.NamespaceDefault
		clusterName      = "test-cluster"

		timeout  = time.Second * 30
		duration = time.Second * 30
		interval = time.Millisecond * 250
	)

	Context("When creating a NodeSet", func() {
		It("Should successfully create nodeset pods", func() {
			By("Creating a new Nodeset")
			nodeset := &slinkyv1alpha1.NodeSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: slinkyv1alpha1.GroupVersion.String(),
					Kind:       slinkyv1alpha1.NodeSetKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        nodesetName,
					Namespace:   nodesetNamespace,
					Labels:      map[string]string{"foo": "bar"},
					Annotations: map[string]string{"biz": "buz"},
				},
				Spec: slinkyv1alpha1.NodeSetSpec{
					ClusterName: clusterName,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
					Replicas: ptr.To[int32](2),
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "pod",
							Namespace:   nodesetNamespace,
							Labels:      map[string]string{"foo": "bar"},
							Annotations: map[string]string{"biz": "buz"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "image-foo",
								},
							},
							Tolerations: []corev1.Toleration{
								{
									// Tolerate this taint when running
									// in test mode as manually added nodes
									// will automatically be tainted
									Key:    "node.kubernetes.io/not-ready",
									Effect: corev1.TaintEffectNoSchedule,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, nodeset)).To(Succeed())

			nodesetLookupKey := types.NamespacedName{Name: nodesetName, Namespace: nodesetNamespace}
			createdNodeset := &slinkyv1alpha1.NodeSet{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, nodesetLookupKey, createdNodeset)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Expect(createdNodeset.Spec.ClusterName).To(Equal("test-cluster"))

			By("Creating Nodeset pods given replica count")
			// Wait for two pods to be created by the NodeSet Controller
			podList := &corev1.PodList{}
			optsList := &k8sclient.ListOptions{
				Namespace:     nodeset.Namespace,
				LabelSelector: labels.Everything(),
			}
			replicas := int(ptr.Deref(nodeset.Spec.Replicas, 0))
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, podList, optsList)).To(Succeed())
				g.Expect(len(podList.Items)).Should(Equal(replicas))
			}, timeout, interval).Should(Succeed())

			By("Simulating Slurm functionality")
			// Simulate Kubernetes marking pods as healthy,
			// and Slurm registering the pods that were just created.
			slurmNodes := make([]slurmtypes.V0041Node, 0)
			for _, pod := range podList.Items {
				// Register Slurm node for pod
				node := slurmtypes.V0041Node{
					V0041Node: v0041.V0041Node{
						Name:  ptr.To(nodesetutils.GetNodeName(&pod)),
						State: ptr.To([]v0041.V0041NodeState{v0041.V0041NodeStateIDLE}),
					},
				}
				slurmNodes = append(slurmNodes, node)
			}

			// Simulate the Cluster controller having added the a slurm-client for the NodeSet.
			// NOTE: we need to do this after we know what the pod are, otherwise Slurm node
			// names will not match.
			slurmClusters.Add(types.NamespacedName{Name: clusterName, Namespace: nodesetNamespace},
				newFakeClientList(interceptor.Funcs{}, &slurmtypes.V0041NodeList{
					Items: slurmNodes,
				}),
			)

			By("Simulating Kubernetes functionality")
			for _, pod := range podList.Items {
				// Mark pod as being healthy
				pod.Status.Phase = corev1.PodRunning
				podCond := corev1.PodCondition{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				}
				pod.Status.Conditions = append(pod.Status.Conditions, podCond)
				Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())
			}

			By("NodeSet scale down")

			// Scale down a NodeSet to verify pods are deleted and
			// Slurm nodes are drained and deleted
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, nodesetLookupKey, createdNodeset)).To(Succeed())
				createdNodeset.Spec.Replicas = ptr.To[int32](0)
				g.Expect(k8sClient.Update(ctx, createdNodeset)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// Verify the Slurm nodes are marked as NodeStateDRAIN
			clusterKey := types.NamespacedName{Namespace: nodesetNamespace, Name: clusterName}
			slurmClient := slurmClusters.Get(clusterKey)
			Eventually(func(g Gomega) {
				slurmNodes := &slurmtypes.V0041NodeList{}
				g.Expect(slurmClient.List(ctx, slurmNodes)).To(Succeed())
				for _, node := range slurmNodes.Items {
					g.Expect(node.GetStateAsSet().Has(v0041.V0041NodeStateDRAIN)).Should(BeTrue())
				}
			}, timeout, interval).Should(Succeed())

			By("Deleting NodeSet")
			Expect(k8sClient.Delete(ctx, createdNodeset)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, nodesetLookupKey, createdNodeset)).ShouldNot(Succeed())
			}, timeout, interval).Should(Succeed())
		})
	})
})

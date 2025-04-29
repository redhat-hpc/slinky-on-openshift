// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v0041 "github.com/SlinkyProject/slurm-client/api/v0041"
	slurmclient "github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
	"github.com/SlinkyProject/slurm-client/pkg/client/interceptor"
	"github.com/SlinkyProject/slurm-client/pkg/object"
	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func newFakeClientList(interceptorFuncs interceptor.Funcs, initObjLists ...object.ObjectList) slurmclient.Client {
	return fake.NewClientBuilder().
		WithLists(initObjLists...).
		WithInterceptorFuncs(interceptorFuncs).
		Build()
}

var _ = Describe("Cluster controller", func() {

	const (
		clusterName      = "test-cluster"
		clusterNamespace = "default"
		slurmSecretName  = "slurm-token-secret"

		timeout  = time.Second * 30
		duration = time.Second * 30
		interval = time.Millisecond * 250
	)

	Context("When updating Cluster status", func() {
		It("Should successfully create create a cluster", func() {

			ctx := context.Background()

			By("By creating a new Cluster")
			cluster := &slinkyv1alpha1.Cluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "slinky.slurm.net/v1alpha1",
					Kind:       "Cluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: clusterNamespace,
				},
				Spec: slinkyv1alpha1.ClusterSpec{
					Token: slinkyv1alpha1.ClusterToken{
						SecretRef: slurmSecretName,
					},
					Server: fake.FakeServer,
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			clusterLookupKey := types.NamespacedName{Name: clusterName, Namespace: clusterNamespace}
			createdCluster := &slinkyv1alpha1.Cluster{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, clusterLookupKey, createdCluster)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Expect(createdCluster.Spec.Token.SecretRef).To(Equal(slurmSecretName))
			Expect(createdCluster.Spec.Server).To(Equal(fake.FakeServer))
			Expect(createdCluster.Status.IsReady).To(BeFalse())

			// Create a secret to match the Cluster CR. This is intentionally added after
			// the Culster CR so the cluster controller attempts to retry reading the secret
			By("By creating the slurm secret")
			slurmSecret := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      slurmSecretName,
					Namespace: clusterNamespace,
				},
				Data: map[string][]byte{
					"auth-token": []byte(fake.FakeSecret),
				},
			}
			Expect(k8sClient.Create(ctx, slurmSecret)).To(Succeed())

			// Wait for slurmCluster client to be added before continuing
			Eventually(func(g Gomega) {
				g.Expect(slurmClusters.Get(types.NamespacedName{Name: clusterName, Namespace: clusterNamespace})).ShouldNot(BeNil())
			}, timeout, interval).Should(Succeed())

			// Replace the slurm client with a fake client that has a
			// successful ping response in the cache
			slurmClusters.Add(types.NamespacedName{Name: clusterName, Namespace: clusterNamespace},
				newFakeClientList(interceptor.Funcs{},
					&slurmtypes.V0041ControllerPingList{
						Items: []slurmtypes.V0041ControllerPing{
							{V0041ControllerPing: v0041.V0041ControllerPing{Hostname: ptr.To("localhost"), Pinged: ptr.To(slurmtypes.V0041ControllerPingPingedUP)}},
						},
					},
				))

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, clusterLookupKey, createdCluster)).To(Succeed())
				g.Expect(createdCluster.Status.IsReady).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// Delete the cluster CR and verify the representative cluster in
			// slurmClusters is removed
			By("By deleting the cluster")
			Expect(k8sClient.Delete(ctx, cluster)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(slurmClusters.Get(types.NamespacedName{Name: clusterName, Namespace: clusterNamespace})).Should(BeNil())
			}, timeout, interval).Should(Succeed())
		})
	})
})

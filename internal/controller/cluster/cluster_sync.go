// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	slurmclient "github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/object"
	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	nodesetcontroller "github.com/SlinkyProject/slurm-operator/internal/controller/nodeset"
	"github.com/SlinkyProject/slurm-operator/internal/utils"
)

// Sync implements control logic for synchronizing a Cluster.
func (r *ClusterReconciler) Sync(ctx context.Context, req reconcile.Request) error {
	logger := log.FromContext(ctx)
	clusterName := req.NamespacedName

	cluster := &slinkyv1alpha1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Cluster has been deleted.", "request", req)
			r.slurmClientDelete(ctx, clusterName)
			return nil
		}
		return err
	}

	// Make a copy now to avoid mutation errors.
	cluster = cluster.DeepCopy()

	if err := r.syncCluster(ctx, cluster); err != nil {
		errors := []error{err}
		if err := r.syncClusterStatus(ctx, cluster); err != nil {
			errors = append(errors, err)
		}
		return utilerrors.NewAggregate(errors)
	}

	return r.syncClusterStatus(ctx, cluster)
}

// syncCluster performs the main syncing logic.
func (r *ClusterReconciler) syncCluster(
	ctx context.Context,
	cluster *slinkyv1alpha1.Cluster,
) error {
	clusterName := types.NamespacedName{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.GetName(),
	}

	// Handle resources marked for deletion
	if cluster.GetDeletionTimestamp() != nil {
		r.slurmClientDelete(ctx, clusterName)
		return nil
	}

	if err := r.slurmClientUpdate(ctx, cluster); err != nil {
		return err
	}

	return nil
}

// slurmClientDelete handles stopping and deleing the cluster/slurmClient data.
func (r *ClusterReconciler) slurmClientDelete(
	ctx context.Context,
	clusterName types.NamespacedName,
) {
	logger := log.FromContext(ctx)

	// Remove slurm client
	if r.SlurmClusters.Remove(clusterName) {
		logger.V(1).Info("Removed slurm cluster client", "clusterName", clusterName.String())
	}
}

// slurmClientUpdate handles updates to the cluster/slurmClient data.
func (r *ClusterReconciler) slurmClientUpdate(
	ctx context.Context,
	cluster *slinkyv1alpha1.Cluster,
) error {
	logger := log.FromContext(ctx)
	clusterName := types.NamespacedName{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.GetName(),
	}

	// Lookup Secret from Cluster reference
	secret := &corev1.Secret{}
	secretName := cluster.Spec.Token.SecretRef
	secretNamespacedName := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      secretName,
	}
	if err := r.Get(ctx, secretNamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Secret not found, retry later", "secretName", secretName)
			durationStore.Push(utils.KeyFunc(cluster), requeueSecretTime)
			r.slurmClientDelete(ctx, clusterName)
			return nil
		}
		logger.Info("Failed to get secret", "secretName", secretName, "error", err)
		return err
	}
	authToken := string(secret.Data["auth-token"])

	// Lookup slurm client
	server := cluster.Spec.Server
	slurmClientOld := r.SlurmClusters.Get(clusterName)

	// Determine if client is unchanged
	if (slurmClientOld != nil) &&
		(slurmClientOld.GetServer() == server) &&
		(slurmClientOld.GetToken() == authToken) {
		return nil
	}

	// Create slurm client
	config := &slurmclient.Config{
		Server:    server,
		AuthToken: authToken,
	}
	options := &slurmclient.ClientOptions{
		DisableFor: []object.Object{
			&slurmtypes.V0041ControllerPing{},
		},
	}
	slurmClient, err := slurmclient.NewClient(config, options)
	if err != nil {
		logger.Error(err, "Failed to create slurm client")
		return nil
	}
	nodesetcontroller.SetEventHandler(slurmClient, r.EventCh)

	// Add slurm client
	if r.SlurmClusters.Add(clusterName, slurmClient) {
		logger.Info("Added slurm cluster client", "clusterName", clusterName.String())
	}

	return nil
}

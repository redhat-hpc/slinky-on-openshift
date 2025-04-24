// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	"github.com/SlinkyProject/slurm-operator/internal/utils"
)

// syncClusterStatus handles determining and updating the cluster status.
func (r *ClusterReconciler) syncClusterStatus(
	ctx context.Context,
	cluster *slinkyv1alpha1.Cluster,
) error {
	logger := log.FromContext(ctx)
	status := cluster.Status.DeepCopy()

	isReady := false
	if ok, err := r.slurmControl.PingController(ctx, cluster); err != nil {
		logger.Error(err, "unable to ping cluster", "cluster", klog.KObj(cluster))
	} else if ok {
		isReady = ok
	}
	status.IsReady = isReady

	if err := r.updateStatus(ctx, cluster, status); err != nil {
		return fmt.Errorf("error updating Cluster(%s) status: %v", klog.KObj(cluster), err)
	}

	if !status.IsReady {
		durationStore.Push(utils.KeyFunc(cluster), requeueReadyTime)
	}

	return nil
}

func (r *ClusterReconciler) updateStatus(
	ctx context.Context,
	cluster *slinkyv1alpha1.Cluster,
	status *slinkyv1alpha1.ClusterStatus,
) error {
	// do not perform an update when the status is consistant
	if !inconsistentStatus(cluster, status) {
		return nil
	}

	// copy cluster and update its status
	cluster = cluster.DeepCopy()
	if err := r.updateClusterStatus(ctx, cluster, status); err != nil {
		return err
	}

	return nil
}

func inconsistentStatus(
	cluster *slinkyv1alpha1.Cluster,
	status *slinkyv1alpha1.ClusterStatus,
) bool {
	return status.IsReady != cluster.Status.IsReady
}

func (r *ClusterReconciler) updateClusterStatus(
	ctx context.Context,
	cluster *slinkyv1alpha1.Cluster,
	newStatus *slinkyv1alpha1.ClusterStatus,
) error {
	logger := log.FromContext(ctx)

	namespacedName := types.NamespacedName{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.GetName(),
	}

	logger.V(1).Info("Pending Cluster Status update",
		"cluster", klog.KObj(cluster), "newStatus", newStatus)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate := &slinkyv1alpha1.Cluster{}
		if err := r.Get(ctx, namespacedName, toUpdate); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		toUpdate.Status = *newStatus
		return r.Status().Update(ctx, toUpdate)
	})
}

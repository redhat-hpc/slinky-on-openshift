// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package nodeset

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	"github.com/SlinkyProject/slurm-operator/internal/utils"
	"github.com/SlinkyProject/slurm-operator/internal/utils/historycontrol"
)

// syncStatus handles synchronizing Slurm Nodes and NodeSet Status.
func (r *NodeSetReconciler) syncStatus(
	ctx context.Context,
	nodeset *slinkyv1alpha1.NodeSet,
	pods []*corev1.Pod,
	currentRevision, updateRevision *appsv1.ControllerRevision,
	collisionCount int32,
	hash string,
	errors ...error,
) error {
	if err := r.syncSlurmStatus(ctx, nodeset, pods); err != nil {
		errors = append(errors, err)
	}

	if err := r.syncNodeSetStatus(ctx, nodeset, pods, currentRevision, updateRevision, collisionCount, hash); err != nil {
		errors = append(errors, err)
	}

	return utilerrors.NewAggregate(errors)
}

// syncSlurmStatus handles synchronizing Slurm Node Status given the pods.
func (r *NodeSetReconciler) syncSlurmStatus(
	ctx context.Context,
	nodeset *slinkyv1alpha1.NodeSet,
	pods []*corev1.Pod,
) error {
	syncSlurmStatusFn := func(i int) error {
		pod := pods[i]
		if !utils.IsHealthy(pod) {
			return nil
		}
		return r.slurmControl.UpdateNodeWithPodInfo(ctx, nodeset, pod)
	}
	if _, err := utils.SlowStartBatch(len(pods), utils.SlowStartInitialBatchSize, syncSlurmStatusFn); err != nil {
		return err
	}

	return nil
}

// syncSlurmStatus handles synchronizing NodeSet Status.
func (r *NodeSetReconciler) syncNodeSetStatus(
	ctx context.Context,
	nodeset *slinkyv1alpha1.NodeSet,
	pods []*corev1.Pod,
	currentRevision, updateRevision *appsv1.ControllerRevision,
	collisionCount int32,
	hash string,
) error {
	logger := log.FromContext(ctx)

	selector, err := metav1.LabelSelectorAsSelector(nodeset.Spec.Selector)
	if err != nil {
		return fmt.Errorf("could not get label selector for NodeSet(%s): %v", klog.KObj(nodeset), err)
	}

	replicaStatus := r.calculateReplicaStatus(nodeset, pods, currentRevision, updateRevision)
	slurmNodeStatus, err := r.slurmControl.CalculateNodeStatus(ctx, nodeset, pods)
	if err != nil {
		return err
	}

	newStatus := &slinkyv1alpha1.NodeSetStatus{
		Replicas:            replicaStatus.Replicas,
		UpdatedReplicas:     replicaStatus.Updated,
		ReadyReplicas:       replicaStatus.Ready,
		AvailableReplicas:   replicaStatus.Available,
		UnavailableReplicas: replicaStatus.Unavailable,
		SlurmIdle:           slurmNodeStatus.Idle,
		SlurmAllocated:      slurmNodeStatus.Allocated + slurmNodeStatus.Mixed,
		SlurmDown:           slurmNodeStatus.Down,
		SlurmDrain:          slurmNodeStatus.Drain,
		ObservedGeneration:  nodeset.Generation,
		NodeSetHash:         hash,
		CollisionCount:      &collisionCount,
		Selector:            selector.String(),
		Conditions:          []metav1.Condition{},
	}
	newStatus.Conditions = append(newStatus.Conditions, nodeset.Status.Conditions...)

	if apiequality.Semantic.DeepEqual(nodeset.Status, newStatus) {
		logger.V(2).Info("NodeSet Status has not changed, skipping status update", "nodeset", klog.KObj(nodeset), "status", nodeset.Status)
		return nil
	}

	if err := r.updateNodeSetStatus(ctx, nodeset, newStatus); err != nil {
		return err
	}

	key := klog.KObj(nodeset).String()
	if nodeset.Spec.MinReadySeconds >= 0 && (newStatus.ReadyReplicas != newStatus.AvailableReplicas) {
		// Resync the NodeSet after MinReadySeconds as a last line of defense to guard against clock-skew.
		durationStore.Push(key, (time.Duration(nodeset.Spec.MinReadySeconds)*time.Second)+time.Second)
	} else if slurmNodeStatus.Total != newStatus.Replicas {
		// Resync the NodeSet until the Slurm counts are correct.
		durationStore.Push(key, 10*time.Second)
	}

	return nil
}

type replicaStatus struct {
	Replicas    int32
	Ready       int32
	Available   int32
	Unavailable int32
	Current     int32
	Updated     int32
}

// calculateReplicaStatus will calculate the status of the given pods.
func (r *NodeSetReconciler) calculateReplicaStatus(
	nodeset *slinkyv1alpha1.NodeSet,
	pods []*corev1.Pod,
	currentRevision, updateRevision *appsv1.ControllerRevision,
) replicaStatus {
	status := replicaStatus{}

	now := metav1.Now()
	for _, pod := range pods {
		// Count the Replicas
		if utils.IsCreated(pod) {
			status.Replicas++
		}
		// Count the Ready and Available replicas
		if utils.IsRunningAndReady(pod) {
			status.Ready++
			if podutil.IsPodAvailable(pod, nodeset.Spec.MinReadySeconds, now) {
				status.Available++
			}
		}
		// Count the Current and Updated replicas
		if utils.IsCreated(pod) && !utils.IsTerminating(pod) {
			podHash := historycontrol.GetRevision(pod.GetLabels())
			curRevHash := historycontrol.GetRevision(currentRevision.GetLabels())
			newRevHash := historycontrol.GetRevision(updateRevision.GetLabels())
			if podHash == curRevHash {
				status.Current++
			}
			if podHash == newRevHash {
				status.Updated++
			}
		}
	}
	// Infer the Unavailable replicas
	status.Unavailable = utils.Clamp(status.Replicas-status.Available, 0, status.Replicas)

	return status
}

// updateNodeSetStatus handles updating the NodeSet status on the Kubernetes API.
// The Status update will be retried on all failures other than NotFound.
func (r *NodeSetReconciler) updateNodeSetStatus(
	ctx context.Context,
	nodeset *slinkyv1alpha1.NodeSet,
	newStatus *slinkyv1alpha1.NodeSetStatus,
) error {
	logger := log.FromContext(ctx)

	namespacedName := types.NamespacedName{
		Namespace: nodeset.GetNamespace(),
		Name:      nodeset.GetName(),
	}

	logger.V(1).Info("Pending NodeSet Status update",
		"nodeset", klog.KObj(nodeset), "newStatus", newStatus)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate := &slinkyv1alpha1.NodeSet{}
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

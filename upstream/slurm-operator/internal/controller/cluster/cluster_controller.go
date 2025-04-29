// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	"github.com/SlinkyProject/slurm-operator/internal/controller/cluster/slurmcontrol"
	"github.com/SlinkyProject/slurm-operator/internal/resources"
	"github.com/SlinkyProject/slurm-operator/internal/utils/durationstore"
)

const (
	// BackoffGCInterval is the time that has to pass before next iteration of backoff GC is run
	BackoffGCInterval = 1 * time.Minute
)

func init() {
	flag.IntVar(&maxConcurrentReconciles, "cluster-workers", maxConcurrentReconciles, "Max concurrent workers for Cluster controller.")
}

var (
	// this is a short cut for any sub-functions to notify the reconcile how long to wait to requeue
	durationStore = durationstore.NewDurationStore(durationstore.Greater)

	maxConcurrentReconciles = 1

	onceBackoffGC     sync.Once
	failedPodsBackoff = flowcontrol.NewBackOff(1*time.Second, 15*time.Minute)
	requeueSecretTime = 10 * time.Second
	requeueReadyTime  = 30 * time.Second
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	SlurmClusters *resources.Clusters
	EventCh       chan event.GenericEvent

	slurmControl  slurmcontrol.SlurmControlInterface
	eventRecorder record.EventRecorderLogger
}

//+kubebuilder:rbac:groups=slinky.slurm.net,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=slinky.slurm.net,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=slinky.slurm.net,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	logger := log.FromContext(ctx)
	logger.Info("Started syncing Cluster", "request", req)

	onceBackoffGC.Do(func() {
		go wait.Until(failedPodsBackoff.GC, BackoffGCInterval, ctx.Done())
	})

	startTime := time.Now()
	defer func() {
		if retErr == nil {
			if res.Requeue || res.RequeueAfter > 0 {
				logger.Info("Finished syncing Cluster", "duration", time.Since(startTime), "result", res)
			} else {
				logger.Info("Finished syncing Cluster", "duration", time.Since(startTime))
			}
		} else {
			logger.Info("Finished syncing Cluster", "duration", time.Since(startTime), "error", retErr)
		}
		// clean the duration store
		_ = durationStore.Pop(req.Namespace)
	}()

	retErr = r.Sync(ctx, req)
	res = reconcile.Result{
		RequeueAfter: durationStore.Pop(req.String()),
	}
	return res, retErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.EventCh == nil {
		return fmt.Errorf("EventCh cannot be nil")
	}
	r.eventRecorder = record.NewBroadcaster().NewRecorder(r.Scheme, corev1.EventSource{Component: "cluster-controller"})
	r.slurmControl = slurmcontrol.NewSlurmControl(r.SlurmClusters)
	return ctrl.NewControllerManagedBy(mgr).
		Named("cluster-controller").
		For(&slinkyv1alpha1.Cluster{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueRequestsForSecrets),
		).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}).
		Complete(r)
}

func (r *ClusterReconciler) enqueueRequestsForSecrets(
	ctx context.Context,
	o client.Object,
) []reconcile.Request {
	requests := make([]reconcile.Request, 0)
	isDeleted := false

	// Lookup secret
	secret := &corev1.Secret{}
	secretNamespacedName := types.NamespacedName{
		Namespace: o.GetNamespace(),
		Name:      o.GetName(),
	}
	if err := r.Get(ctx, secretNamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			isDeleted = true
		}
	}

	// Lookup cluster
	clusterList := &slinkyv1alpha1.ClusterList{}
	if err := r.List(ctx, clusterList); err != nil {
		if apierrors.IsNotFound(err) {
			return requests
		}
		return requests
	}

	// Queue a cluster request when the referenced secret was changed or deleted
	for _, cluster := range clusterList.Items {
		if !isDeleted &&
			((o.GetNamespace() != cluster.GetNamespace()) ||
				(o.GetName() != cluster.Spec.Token.SecretRef)) {
			continue
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: cluster.GetNamespace(),
				Name:      cluster.GetName(),
			},
		})
	}

	return requests
}

// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package slurmcontrol

import (
	"context"
	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slurmclient "github.com/SlinkyProject/slurm-client/pkg/client"
	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	"github.com/SlinkyProject/slurm-operator/internal/resources"
)

type SlurmControlInterface interface {
	// PingController sends a ping request to check connectivity.
	PingController(ctx context.Context, cluster *slinkyv1alpha1.Cluster) (bool, error)
}

// realSlurmControl is the default implementation of SlurmControlInterface.
type realSlurmControl struct {
	slurmClusters *resources.Clusters
}

// PingController implements SlurmControlInterface.
func (r *realSlurmControl) PingController(ctx context.Context, cluster *slinkyv1alpha1.Cluster) (bool, error) {
	logger := log.FromContext(ctx)

	slurmClient := r.lookupClient(cluster)
	if slurmClient == nil {
		logger.V(2).Info("no client for cluster, cannot do PingController()",
			"cluster", klog.KObj(cluster))
		return false, nil
	}

	pingList := &slurmtypes.V0041ControllerPingList{}
	if err := slurmClient.List(ctx, pingList); err != nil {
		if tolerateError(err) {
			return false, nil
		}
		return false, err
	}
	for _, ping := range pingList.Items {
		if ptr.Deref(ping.Pinged, "") == slurmtypes.V0041ControllerPingPingedUP {
			return true, nil
		}
	}

	return false, nil
}

func (r *realSlurmControl) lookupClient(cluster *slinkyv1alpha1.Cluster) slurmclient.Client {
	clusterName := types.NamespacedName{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.GetName(),
	}
	return r.slurmClusters.Get(clusterName)
}

var _ SlurmControlInterface = &realSlurmControl{}

func NewSlurmControl(clusters *resources.Clusters) SlurmControlInterface {
	return &realSlurmControl{
		slurmClusters: clusters,
	}
}

func tolerateError(err error) bool {
	if err == nil {
		return true
	}
	errText := err.Error()
	if errText == http.StatusText(http.StatusNotFound) ||
		errText == http.StatusText(http.StatusNoContent) {
		return true
	}
	return false
}

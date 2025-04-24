// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package slurmcontrol

import (
	"context"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/puttsk/hostlist"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v0041 "github.com/SlinkyProject/slurm-client/api/v0041"
	slurmclient "github.com/SlinkyProject/slurm-client/pkg/client"
	slurmobject "github.com/SlinkyProject/slurm-client/pkg/object"
	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"

	slinkyv1alpha1 "github.com/SlinkyProject/slurm-operator/api/v1alpha1"
	nodesetutils "github.com/SlinkyProject/slurm-operator/internal/controller/nodeset/utils"
	"github.com/SlinkyProject/slurm-operator/internal/resources"
	"github.com/SlinkyProject/slurm-operator/internal/utils/podinfo"
	"github.com/SlinkyProject/slurm-operator/internal/utils/timestore"
)

type SlurmControlInterface interface {
	// UpdateNodeWithPodInfo handles updating the Node with its pod info
	UpdateNodeWithPodInfo(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod) error
	// MakeNodeDrain handles adding the DRAIN state to the slurm node.
	MakeNodeDrain(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod, reason string) error
	// MakeNodeUndrain handles removing the DRAIN state from the slurm node.
	MakeNodeUndrain(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod, reason string) error
	// IsNodeDrain checks if the slurm node has the DRAIN state.
	IsNodeDrain(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod) (bool, error)
	// IsNodeDrained checks if the slurm node is drained.
	IsNodeDrained(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod) (bool, error)
	// CalculateNodeStatus returns the current state of the registered slurm nodes.
	CalculateNodeStatus(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pods []*corev1.Pod) (SlurmNodeStatus, error)
	// GetNodeDeadlines returns a map of node to its deadline time.Time calculated from running jobs.
	GetNodeDeadlines(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pods []*corev1.Pod) (*timestore.TimeStore, error)
}

// realSlurmControl is the default implementation of SlurmControlInterface.
type realSlurmControl struct {
	slurmClusters *resources.Clusters
}

// GetNodeNames implements SlurmControlInterface.
func (r *realSlurmControl) GetNodeNames(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pods []*corev1.Pod) ([]string, error) {
	logger := log.FromContext(ctx)

	slurmClient := r.lookupClient(nodeset)
	if slurmClient == nil {
		logger.V(2).Info("no client for nodeset, cannot do GetNodeNames()",
			"nodeset", klog.KObj(nodeset))
		return nil, nil
	}

	nodeList := &slurmtypes.V0041NodeList{}
	if err := slurmClient.List(ctx, nodeList); err != nil {
		return nil, err
	}

	podNodeNameSet := set.New[string]()
	for _, pod := range pods {
		podNodeName := nodesetutils.GetNodeName(pod)
		podNodeNameSet.Insert(podNodeName)
	}

	nodeNames := []string{}
	for _, node := range nodeList.Items {
		nodeName := ptr.Deref(node.Name, "")
		if !podNodeNameSet.Has(nodeName) {
			continue
		}
		nodeNames = append(nodeNames, nodeName)
	}

	return nodeNames, nil
}

// UpdateNodeWithPodInfo implements SlurmControlInterface.
func (r *realSlurmControl) UpdateNodeWithPodInfo(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)

	slurmClient := r.lookupClient(nodeset)
	if slurmClient == nil {
		logger.V(2).Info("no client for nodeset, cannot do UpdateNodeWithPodInfo()",
			"nodeset", klog.KObj(nodeset), "pod", klog.KObj(pod))
		return nil
	}

	slurmNode := &slurmtypes.V0041Node{}
	key := slurmobject.ObjectKey(nodesetutils.GetNodeName(pod))
	if err := slurmClient.Get(ctx, key, slurmNode); err != nil {
		if tolerateError(err) {
			return nil
		}
		return err
	}

	podInfo := podinfo.PodInfo{
		Namespace: pod.GetNamespace(),
		PodName:   pod.GetName(),
	}
	podInfoOld := &podinfo.PodInfo{}
	_ = podinfo.ParseIntoPodInfo(slurmNode.Comment, podInfoOld)

	if podInfoOld.Equal(podInfo) {
		logger.V(3).Info("Node already contains podInfo, skipping update request",
			"node", slurmNode.GetKey(), "podInfo", podInfo)
		return nil
	}

	logger.Info("Update Slurm Node with Kubernetes Pod info",
		"Node", slurmNode.Name, "podInfo", podInfo)
	req := v0041.V0041UpdateNodeMsg{
		Comment: ptr.To(podInfo.ToString()),
	}
	if err := slurmClient.Update(ctx, slurmNode, req); err != nil {
		if tolerateError(err) {
			return nil
		}
		return err
	}

	return nil
}

const nodeReasonPrefix = "slurm-operator:"

// MakeNodeDrain implements SlurmControlInterface.
func (r *realSlurmControl) MakeNodeDrain(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod, reason string) error {
	logger := log.FromContext(ctx)

	slurmClient := r.lookupClient(nodeset)
	if slurmClient == nil {
		logger.V(2).Info("no client for nodeset, cannot do MakeNodeDrain()",
			"nodeset", klog.KObj(nodeset), "pod", klog.KObj(pod))
		return nil
	}

	slurmNode := &slurmtypes.V0041Node{}
	key := slurmobject.ObjectKey(nodesetutils.GetNodeName(pod))
	if err := slurmClient.Get(ctx, key, slurmNode); err != nil {
		if tolerateError(err) {
			return nil
		}
		return err
	}

	logger.V(1).Info("make slurm node drain",
		"nodeset", klog.KObj(nodeset), "pod", klog.KObj(pod))
	req := v0041.V0041UpdateNodeMsg{
		State:  ptr.To([]v0041.V0041UpdateNodeMsgState{v0041.V0041UpdateNodeMsgStateDRAIN}),
		Reason: ptr.To(nodeReasonPrefix + " " + reason),
	}
	if err := slurmClient.Update(ctx, slurmNode, req); err != nil {
		if tolerateError(err) {
			return nil
		}
		return err
	}

	return nil
}

// MakeNodeUndrain implements SlurmControlInterface.
func (r *realSlurmControl) MakeNodeUndrain(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod, reason string) error {
	logger := log.FromContext(ctx)

	slurmClient := r.lookupClient(nodeset)
	if slurmClient == nil {
		logger.V(2).Info("no client for nodeset, cannot do MakeNodeUndrain()",
			"nodeset", klog.KObj(nodeset), "pod", klog.KObj(pod))
		return nil
	}

	slurmNode := &slurmtypes.V0041Node{}
	key := slurmobject.ObjectKey(nodesetutils.GetNodeName(pod))
	if err := slurmClient.Get(ctx, key, slurmNode); err != nil {
		if tolerateError(err) {
			return nil
		}
		return err
	}

	nodeReason := ptr.Deref(slurmNode.Reason, "")
	if !slurmNode.GetStateAsSet().Has(v0041.V0041NodeStateDRAIN) ||
		slurmNode.GetStateAsSet().Has(v0041.V0041NodeStateUNDRAIN) {
		logger.V(1).Info("Node is already undrained, skipping undrain request",
			"node", slurmNode.GetKey(), "nodeState", slurmNode.State)
		return nil
	} else if nodeReason != "" && !strings.Contains(nodeReason, nodeReasonPrefix) {
		logger.Info("Node was drained but not by slurm-operator, skipping undrain request",
			"node", slurmNode.GetKey(), "nodeReason", nodeReason)
		return nil
	}

	logger.V(1).Info("make slurm node undrain",
		"nodeset", klog.KObj(nodeset), "pod", klog.KObj(pod))
	req := v0041.V0041UpdateNodeMsg{
		State:  ptr.To([]v0041.V0041UpdateNodeMsgState{v0041.V0041UpdateNodeMsgStateUNDRAIN}),
		Reason: ptr.To(nodeReasonPrefix + " " + reason),
	}
	if err := slurmClient.Update(ctx, slurmNode, req); err != nil {
		if tolerateError(err) {
			return nil
		}
		return err
	}

	return nil
}

// IsNodeDrain implements SlurmControlInterface.
func (r *realSlurmControl) IsNodeDrain(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod) (bool, error) {
	logger := log.FromContext(ctx)

	slurmClient := r.lookupClient(nodeset)
	if slurmClient == nil {
		logger.V(2).Info("no client for nodeset, cannot do IsNodeDrain()",
			"nodeset", klog.KObj(nodeset), "pod", klog.KObj(pod))
		return true, nil
	}

	slurmNode := &slurmtypes.V0041Node{}
	key := slurmobject.ObjectKey(nodesetutils.GetNodeName(pod))
	if err := slurmClient.Get(ctx, key, slurmNode); err != nil {
		if tolerateError(err) {
			return true, nil
		}
		return false, err
	}

	isDrain := slurmNode.GetStateAsSet().Has(v0041.V0041NodeStateDRAIN)
	return isDrain, nil
}

// IsNodeDrained implements SlurmControlInterface.
func (r *realSlurmControl) IsNodeDrained(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pod *corev1.Pod) (bool, error) {
	logger := log.FromContext(ctx)

	slurmClient := r.lookupClient(nodeset)
	if slurmClient == nil {
		logger.V(2).Info("no client for nodeset, cannot do IsNodeDrained()",
			"nodeset", klog.KObj(nodeset), "pod", klog.KObj(pod))
		return true, nil
	}

	slurmNode := &slurmtypes.V0041Node{}
	key := slurmobject.ObjectKey(nodesetutils.GetNodeName(pod))
	if err := slurmClient.Get(ctx, key, slurmNode); err != nil {
		if tolerateError(err) {
			return true, nil
		}
		return false, err
	}

	// DRAINED = IDLE+DRAIN || DOWN+DRAIN
	baseState := slurmNode.GetStateAsSet().HasAny(v0041.V0041NodeStateIDLE, v0041.V0041NodeStateDOWN)
	flagState := slurmNode.GetStateAsSet().Has(v0041.V0041NodeStateDRAIN)
	isDrained := baseState && flagState

	return isDrained, nil
}

type SlurmNodeStatus struct {
	Total int32

	// Base State
	Allocated int32
	Down      int32
	Error     int32
	Future    int32
	Idle      int32
	Mixed     int32
	Unknown   int32

	// Flag State
	Completing    int32
	Drain         int32
	Fail          int32
	Invalid       int32
	InvalidReg    int32
	Maintenance   int32
	NotResponding int32
	Undrain       int32
}

// CalculateNodeStatus implements SlurmControlInterface.
func (r *realSlurmControl) CalculateNodeStatus(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pods []*corev1.Pod) (SlurmNodeStatus, error) {
	logger := log.FromContext(ctx)
	status := SlurmNodeStatus{}

	slurmClient := r.lookupClient(nodeset)
	if slurmClient == nil {
		logger.V(2).Info("no client for nodeset, cannot do CalculateNodeStatus()",
			"nodeset", klog.KObj(nodeset))
		return status, nil
	}

	nodeList := &slurmtypes.V0041NodeList{}
	opts := &slurmclient.ListOptions{RefreshCache: true}
	if err := slurmClient.List(ctx, nodeList, opts); err != nil {
		if tolerateError(err) {
			return status, nil
		}
		return status, err
	}

	podNodeNameSet := set.New[string]()
	for _, pod := range pods {
		podNodeName := nodesetutils.GetNodeName(pod)
		podNodeNameSet.Insert(podNodeName)
	}

	for _, node := range nodeList.Items {
		nodeName := ptr.Deref(node.Name, "")
		if !podNodeNameSet.Has(nodeName) {
			continue
		}
		status.Total++
		// Slurm Node Base States
		switch {
		case node.GetStateAsSet().Has(v0041.V0041NodeStateALLOCATED):
			status.Allocated++
		case node.GetStateAsSet().Has(v0041.V0041NodeStateDOWN):
			status.Down++
		case node.GetStateAsSet().Has(v0041.V0041NodeStateERROR):
			status.Error++
		case node.GetStateAsSet().Has(v0041.V0041NodeStateFUTURE):
			status.Future++
		case node.GetStateAsSet().Has(v0041.V0041NodeStateIDLE):
			status.Idle++
		case node.GetStateAsSet().Has(v0041.V0041NodeStateMIXED):
			status.Mixed++
		case node.GetStateAsSet().Has(v0041.V0041NodeStateUNKNOWN):
			status.Unknown++
		}
		// Slurm Node Flag State
		if node.GetStateAsSet().Has(v0041.V0041NodeStateCOMPLETING) {
			status.Completing++
		}
		if node.GetStateAsSet().Has(v0041.V0041NodeStateDRAIN) {
			status.Drain++
		}
		if node.GetStateAsSet().Has(v0041.V0041NodeStateFAIL) {
			status.Fail++
		}
		if node.GetStateAsSet().Has(v0041.V0041NodeStateINVALID) {
			status.Invalid++
		}
		if node.GetStateAsSet().Has(v0041.V0041NodeStateINVALIDREG) {
			status.InvalidReg++
		}
		if node.GetStateAsSet().Has(v0041.V0041NodeStateMAINTENANCE) {
			status.Maintenance++
		}
		if node.GetStateAsSet().Has(v0041.V0041NodeStateNOTRESPONDING) {
			status.NotResponding++
		}
		if node.GetStateAsSet().Has(v0041.V0041NodeStateUNDRAIN) {
			status.Undrain++
		}
	}

	return status, nil
}

const infiniteDuration = time.Duration(math.MaxInt64)

// GetNodeDeadlines implements SlurmControlInterface.
func (r *realSlurmControl) GetNodeDeadlines(ctx context.Context, nodeset *slinkyv1alpha1.NodeSet, pods []*corev1.Pod) (*timestore.TimeStore, error) {
	logger := log.FromContext(ctx)
	ts := timestore.NewTimeStore(timestore.Greater)

	slurmClient := r.lookupClient(nodeset)
	if slurmClient == nil {
		logger.V(2).Info("no client for nodeset, cannot do GetNodeDeadlines()",
			"nodeset", klog.KObj(nodeset))
		return ts, nil
	}

	slurmNodeNamesSet := set.New[string]()
	for _, pod := range pods {
		slurmNodeName := nodesetutils.GetNodeName(pod)
		slurmNodeNamesSet.Insert(slurmNodeName)
	}

	jobList := &slurmtypes.V0041JobInfoList{}
	if err := slurmClient.List(ctx, jobList); err != nil {
		return nil, err
	}

	for _, job := range jobList.Items {
		if !job.GetStateAsSet().Has(v0041.V0041JobInfoJobStateRUNNING) {
			continue
		}
		slurmNodeNames, err := hostlist.Expand(ptr.Deref(job.Nodes, ""))
		if err != nil {
			logger.Error(err, "failed to expand job node hostlist",
				"job", ptr.Deref(job.JobId, 0))
			return nil, err
		}
		if !slurmNodeNamesSet.HasAny(slurmNodeNames...) {
			continue
		}

		// Get startTime, when the job was launched on the compute node.
		startTime_NoVal := ptr.Deref(job.StartTime, v0041.V0041Uint64NoValStruct{})
		startTime := time.Unix(ptr.Deref(startTime_NoVal.Number, 0), 0)
		// Get the timeLimit, the wall time of the job.
		timeLimit_NoVal := ptr.Deref(job.TimeLimit, v0041.V0041Uint32NoValStruct{})
		timeLimit := time.Duration(ptr.Deref(timeLimit_NoVal.Number, 0)) * time.Minute
		if ptr.Deref(timeLimit_NoVal.Infinite, false) {
			timeLimit = infiniteDuration
		}

		// Push time/duration into the fancy map for each node allocated to the job.
		for _, slurmNodeName := range slurmNodeNames {
			ts.Push(slurmNodeName, startTime.Add(timeLimit))
		}
	}

	return ts, nil
}

func (r *realSlurmControl) lookupClient(nodeset *slinkyv1alpha1.NodeSet) slurmclient.Client {
	clusterName := types.NamespacedName{
		Namespace: nodeset.GetNamespace(),
		Name:      nodeset.Spec.ClusterName,
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

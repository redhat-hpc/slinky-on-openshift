// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// log is for logging in this package.
var nodesetlog = logf.Log.WithName("nodeset-resource")

func (r *NodeSet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-slinky-slurm-net-v1alpha1-nodeset,mutating=true,failurePolicy=fail,sideEffects=None,groups=slinky.slurm.net,resources=nodesets,verbs=create;update,versions=v1alpha1,name=mnodeset.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &NodeSet{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NodeSet) Default(ctx context.Context, obj runtime.Object) error {
	nodeset := obj.(*NodeSet)
	nodesetlog.Info("default", "nodeset", klog.KObj(nodeset))

	if nodeset.Spec.RevisionHistoryLimit == nil {
		nodeset.Spec.RevisionHistoryLimit = ptr.To[int32](0)
	}
	if nodeset.Spec.UpdateStrategy.Type == "" {
		nodeset.Spec.UpdateStrategy.Type = RollingUpdateNodeSetStrategyType
	}
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-slinky-slurm-net-v1alpha1-nodeset,mutating=false,failurePolicy=fail,sideEffects=None,groups=slinky.slurm.net,resources=nodesets,verbs=create;update,versions=v1alpha1,name=vnodeset.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &NodeSet{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeSet) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	nodeset := obj.(*NodeSet)
	nodesetlog.Info("validate create", "nodeset", klog.KObj(nodeset))

	warns, errs := validateNodeSet(nodeset)

	return warns, utilerrors.NewAggregate(errs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeSet) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	newNodeSet := newObj.(*NodeSet)
	oldNodeSet := oldObj.(*NodeSet)
	nodesetlog.Info("validate update", "newNodeSet", klog.KObj(newNodeSet), "oldNodeSet", klog.KObj(oldNodeSet))

	warns, errs := validateNodeSet(newNodeSet)

	updateFields := []string{
		"MinReadySeconds",
		"PersistentVolumeClaimRetentionPolicy",
		"Replicas",
		"RevisionHistoryLimit",
		"Selector",
		"UpdateStrategy",
		"VolumeClaimTemplates",
		"Volumes",
	}
	sort.Strings(updateFields)
	errMsgStub := fmt.Sprintf("Mutatable fields include: %s", strings.Join(updateFields, ", "))
	if newNodeSet.Spec.ClusterName != oldNodeSet.Spec.ClusterName {
		errs = append(errs, fmt.Errorf("updates to `NodeSet.Spec.ClusterName` is forbidden. %v", errMsgStub))
	}
	if newNodeSet.Spec.ServiceName != oldNodeSet.Spec.ServiceName {
		errs = append(errs, fmt.Errorf("updates to `NodeSet.Spec.ServiceName` is forbidden. %v", errMsgStub))
	}

	return warns, utilerrors.NewAggregate(errs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NodeSet) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	nodeset := obj.(*NodeSet)
	nodesetlog.Info("validate delete", "nodeset", klog.KObj(nodeset))

	return nil, nil
}

func validateNodeSet(r *NodeSet) (admission.Warnings, []error) {
	var warns admission.Warnings
	var errs []error

	if r.Spec.ServiceName == "" {
		errs = append(errs, fmt.Errorf("`NodeSet.Spec.ServiceName` cannot be empty"))
	}

	switch r.Spec.UpdateStrategy.Type {
	case RollingUpdateNodeSetStrategyType:
		// valid
	case OnDeleteNodeSetStrategyType:
		// valid
	default:
		errs = append(errs, fmt.Errorf("`NodeSet.Spec.UpdateStrategy.Type` is not valid. Got: %v. Expected of: %s; %s",
			r.Spec.UpdateStrategy.Type, RollingUpdateNodeSetStrategyType, OnDeleteNodeSetStrategyType))
	}

	if r.Spec.PersistentVolumeClaimRetentionPolicy != nil {
		switch r.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted {
		case RetainPersistentVolumeClaimRetentionPolicyType:
			// valid
		case DeletePersistentVolumeClaimRetentionPolicyType:
			// valid
		default:
			errs = append(errs, fmt.Errorf("`NodeSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted` is not valid. Got: %v. Expected of: %s; %s",
				r.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted, RetainPersistentVolumeClaimRetentionPolicyType, DeletePersistentVolumeClaimRetentionPolicyType))
		}
		switch r.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled {
		case RetainPersistentVolumeClaimRetentionPolicyType:
			// valid
		case DeletePersistentVolumeClaimRetentionPolicyType:
			// valid
		default:
			errs = append(errs, fmt.Errorf("`NodeSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled` is not valid. Got: %v. Expected of: %s; %s",
				r.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled, RetainPersistentVolumeClaimRetentionPolicyType, DeletePersistentVolumeClaimRetentionPolicyType))
		}
	}

	return warns, errs
}

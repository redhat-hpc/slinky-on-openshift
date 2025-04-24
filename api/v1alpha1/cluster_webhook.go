// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// log is for logging in this package.
var clusterlog = logf.Log.WithName("cluster-resource")

func (r *Cluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-slinky-slurm-net-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=slinky.slurm.net,resources=clusters,verbs=create;update,versions=v1alpha1,name=mcluster.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &Cluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Cluster) Default(ctx context.Context, obj runtime.Object) error {
	cluster := obj.(*Cluster)
	clusterlog.Info("default", "cluster", klog.KObj(cluster))
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-slinky-slurm-net-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=slinky.slurm.net,resources=clusters,verbs=create;update,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &Cluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cluster := obj.(*Cluster)
	clusterlog.Info("validate create", "cluster", klog.KObj(cluster))

	warns, errs := validateCluster(cluster)

	return warns, utilerrors.NewAggregate(errs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	newCluster := newObj.(*Cluster)
	oldCluster := oldObj.(*Cluster)
	clusterlog.Info("validate update", "newCluster", klog.KObj(newCluster), "oldCluster", klog.KObj(oldCluster))

	warns, errs := validateCluster(newCluster)

	return warns, utilerrors.NewAggregate(errs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cluster := obj.(*Cluster)
	clusterlog.Info("validate delete", "cluster", klog.KObj(cluster))

	return nil, nil
}

func validateCluster(r *Cluster) (admission.Warnings, []error) {
	var warns admission.Warnings
	var errs []error

	if r.Spec.Server == "" {
		errs = append(errs, fmt.Errorf("`Cluster.Spec.Server` cannot be empty"))
	}
	if r.Spec.Token.SecretRef == "" {
		errs = append(errs, fmt.Errorf("`Cluster.Spec.Token.SecretRef` cannot be empty"))
	}

	return warns, errs
}

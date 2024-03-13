/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	"github.com/go-logr/logr"
	quotav1 "github.com/openshift/api/quota/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureQuota ensures Quota presence
func (r *PaasReconciler) EnsureQuota(
	ctx context.Context,
	paas *v1alpha1.Paas,
	quota *quotav1.ClusterResourceQuota,
) error {
	// See if quota already exists and create if it doesn't
	found := &quotav1.ClusterResourceQuota{}
	err := r.Get(ctx, types.NamespacedName{
		Name: quota.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the quota
		if err = r.Create(ctx, quota); err != nil {
			// creating the quota failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, quota, err.Error())
			return err
		} else {
			// creating the quota was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, quota, "succeeded")
			return nil
		}
	} else if err != nil {
		// Error that isn't due to the quota not existing
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, quota, err.Error())
		return err
	} else {
		// Update the quota
		found.Spec = quota.Spec
		if err = r.Update(ctx, found); err != nil {
			// updating the quota failed
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, quota, err.Error())
			return err
		} else {
			// updating the quota was successful
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, quota, "succeeded")
			return nil
		}
	}
}

// backendQuota is a code for Creating Quota
func (r *PaasReconciler) backendQuota(
	ctx context.Context,
	paas *v1alpha1.Paas, suffix string,
	hardQuotas map[corev1.ResourceName]resourcev1.Quantity,
) *quotav1.ClusterResourceQuota {
	var quotaName string
	if suffix == "" {
		quotaName = paas.ObjectMeta.Name
	} else {
		quotaName = fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, suffix)
	}
	logger := getLogger(ctx, paas, "Quota", quotaName)
	logger.Info("Defining quota")
	//matchLabels := map[string]string{"dcs.itsmoplosgroep": paas.Name}
	quota := &quotav1.ClusterResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterResourceQuota",
			APIVersion: "quota.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   quotaName,
			Labels: paas.ClonedLabels(),
		},
		Spec: quotav1.ClusterResourceQuotaSpec{
			Selector: quotav1.ClusterResourceQuotaSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						getConfig().QuotaLabel: quotaName},
				},
			},
			Quota: corev1.ResourceQuotaSpec{
				Hard: hardQuotas,
			},
		},
	}

	logger.Info("Setting owner")
	controllerutil.SetControllerReference(paas, quota, r.Scheme)
	return quota
}

func (r *PaasReconciler) BackendEnabledQuotas(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (quotas []*quotav1.ClusterResourceQuota) {
	quotas = append(quotas, r.backendQuota(ctx, paas, "", paas.Spec.Quota))
	for name, cap := range paas.Spec.Capabilities.AsMap() {
		if cap.IsEnabled() {
			defaults := getConfig().DefaultQuota(
				cap.CapabilityName())
			quota := cap.Quotas().QuotaWithDefaults(
				defaults)
			quotas = append(quotas,
				r.backendQuota(ctx, paas, name, quota))
		}
	}
	return quotas
}

func (r *PaasReconciler) BackendEnabledQuotaStatus(
	paas *v1alpha1.Paas,
) (quotas map[string]v1alpha1.PaasQuotas) {
	quotas = make(map[string]v1alpha1.PaasQuotas)
	quotas["default"] = paas.Spec.Quota
	for name, cap := range paas.Spec.Capabilities.AsMap() {
		if cap.IsEnabled() {
			defaults := getConfig().DefaultQuota(
				cap.CapabilityName())
			quota := cap.Quotas().QuotaWithDefaults(
				defaults)
			quotas[name] = quota
		}
	}
	return quotas
}

func (r *PaasReconciler) BackendDisabledQuotas(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (quotas []string) {
	for name, cap := range paas.Spec.Capabilities.AsMap() {
		if !cap.IsEnabled() {
			quotas = append(quotas, fmt.Sprintf("%s-%s", paas.Name, name))
		}
	}
	return quotas
}

func (r *PaasReconciler) FinalizeClusterQuota(ctx context.Context, paas *v1alpha1.Paas, quotaName string) error {
	logger := getLogger(ctx, paas, "Quota", quotaName)
	logger.Info("Finalizing")
	obj := &quotav1.ClusterResourceQuota{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: quotaName,
	}, obj); err != nil && errors.IsNotFound(err) {
		logger.Info("Does not exist")
		return nil
	} else if err != nil {
		logger.Info("Error retrieving info: " + err.Error())
		return err
	} else {
		logger.Info("Deleting")
		return r.Delete(ctx, obj)
	}
}

func (r *PaasReconciler) FinalizeClusterQuotas(ctx context.Context, paas *v1alpha1.Paas) error {
	suffixes := []string{
		"",
	}
	for name := range paas.Spec.Capabilities.AsMap() {
		suffixes = append(suffixes, name)
	}

	var err error
	for _, suffix := range suffixes {
		quotaName := fmt.Sprintf("%s%s", paas.Name, suffix)
		if cleanErr := r.FinalizeClusterQuota(ctx, paas, quotaName); cleanErr != nil {
			err = cleanErr
		}
	}
	return err
}

func (r *PaasReconciler) ReconcileQuotas(
	ctx context.Context,
	paas *v1alpha1.Paas,
	logger logr.Logger,
) error {
	logger.Info("Creating quotas for PAAS object ")
	// Create quotas if needed
	for _, q := range r.BackendEnabledQuotas(ctx, paas) {
		logger.Info("Creating quota " + q.Name + " for PAAS object ")
		if err := r.EnsureQuota(ctx, paas, q); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while creating quota %s", q.ObjectMeta.Name))
			return err
		}
	}
	paas.Status.Quota = r.BackendEnabledQuotaStatus(paas)
	for _, name := range r.BackendDisabledQuotas(ctx, paas) {
		logger.Info("Cleaning quota " + name + " for PAAS object ")
		if err := r.FinalizeClusterQuota(ctx, paas, name); err != nil {
			logger.Error(err, fmt.Sprintf("Failure while finalizing quota %s", name))
			return err
		}
	}
	return nil
}

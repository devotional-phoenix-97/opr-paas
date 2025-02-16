/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/logging"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *PaasReconciler) GetPaasNs(ctx context.Context, paas *v1alpha1.Paas, name string,
	groups []string, secrets map[string]string,
) (*v1alpha1.PaasNS, error) {
	pns := &v1alpha1.PaasNS{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PaasNS",
			APIVersion: "cpet.belastingdienst.nl/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: paas.Name,
			Labels:    paas.ClonedLabels(),
		},
		Spec: v1alpha1.PaasNSSpec{
			Paas:       paas.Name,
			Groups:     groups,
			SSHSecrets: secrets,
		},
	}
	logger := log.Ctx(ctx)
	logger.Info().Msg("defining")
	pns.ObjectMeta.Labels[config.GetConfig().RequestorLabel] = paas.Spec.Requestor

	logger.Info().Msg("setting Owner")

	if err := controllerutil.SetControllerReference(paas, pns, r.Scheme); err != nil {
		return pns, err
	}
	return pns, nil
}

func (r *PaasReconciler) ensurePaasNs(ctx context.Context, paas *v1alpha1.Paas, pns *v1alpha1.PaasNS) error {
	logger := log.Ctx(ctx)
	logger.Info().Msg("ensuring")

	// See if namespace exists and create if it doesn't
	found := &v1alpha1.PaasNS{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      pns.Name,
		Namespace: pns.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		if err = r.Create(ctx, pns); err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	} else if !paas.AmIOwner(found.OwnerReferences) {
		if err := controllerutil.SetControllerReference(paas, found, r.Scheme); err != nil {
			return err
		}
	}

	found.Spec.Paas = pns.Spec.Paas
	found.Spec.Groups = pns.Spec.Groups
	found.Spec.SSHSecrets = pns.Spec.SSHSecrets
	found.ObjectMeta.Labels = pns.ObjectMeta.Labels
	logger.Info().Str("PaasNs", pns.Name).Msg("updating PaasNs")
	return r.Update(ctx, found)
}

func (r *PaasReconciler) FinalizePaasNss(ctx context.Context, paas *v1alpha1.Paas) error {
	logger := log.Ctx(ctx)
	logger.Info().Msg("finalizing")

	enabledNs := paas.AllEnabledNamespaces()

	// Loop through all namespaces and remove when not should be
	pnsList := &v1alpha1.PaasNSList{}
	if err := r.List(ctx, pnsList, &client.ListOptions{Namespace: paas.Name}); err != nil {
		return err
	}

	for _, pns := range pnsList.Items {
		// Reassign to make sure to use the referenced address of the current iteration
		pns := pns

		if !paas.AmIOwner(pns.OwnerReferences) {
			// logger.Info().Msg("skipping finalization", "Namespace", ns.Name, "Reason", "I am not owner")
		} else if _, isEnabled := enabledNs[pns.Name]; isEnabled {
			// logger.Info().Msg("skipping finalization", "Namespace", ns.Name, "Reason", "Should be there")
		} else if err := r.Delete(ctx, &pns); err != nil {
			return err
		}
	}
	return nil
}

func (r *PaasReconciler) ReconcilePaasNss(
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "paasns")
	logger.Info().Msg("creating default namespace to hold PaasNs resources for Paas object")
	if ns, err := BackendNamespace(ctx, paas, paas.Name, paas.Name, r.Scheme); err != nil {
		logger.Err(err).Msgf("failure while defining namespace %s", paas.Name)
		return err
	} else if err = EnsureNamespace(r.Client, ctx, paas, ns, r.Scheme); err != nil {
		logger.Err(err).Msgf("failure while creating namespace %s", paas.Name)
		return err
	}
	logger.Info().Msg("creating PaasNs resources for Paas object")
	for nsName := range paas.AllEnabledNamespaces() {
		pns, err := r.GetPaasNs(ctx, paas, nsName, paas.Spec.Groups.Keys(), paas.GetNsSSHSecrets(nsName))
		if err != nil {
			logger.Err(err).Msgf("failure while creating PaasNs %s",
				types.NamespacedName{Name: pns.Name, Namespace: pns.Namespace})
			return err
		}
		if err = r.ensurePaasNs(ctx, paas, pns); err != nil {
			logger.Err(err).Msgf("failure while creating PaasNs %s",
				types.NamespacedName{Name: pns.Name, Namespace: pns.Namespace})
			return err
		}
	}

	logger.Info().Msg("cleaning obsolete namespaces ")
	if err := r.FinalizePaasNss(ctx, paas); err != nil {
		return err
	}

	return nil
}

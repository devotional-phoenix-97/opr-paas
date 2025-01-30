/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupPaasNsWebhookWithManager registers the webhook for PaasNs in the manager.
func SetupPaasNsWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&PaasNS{}).
		WithValidator(&PaasNSCustomValidator{}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-cpet-belastingdienst-nl-v1alpha1-paasns,mutating=false,failurePolicy=fail,sideEffects=None,groups=cpet.belastingdienst.nl,resources=paasns,verbs=create;update,versions=v1alpha1,name=vpaasns-v1alpha1.kb.io,admissionReviewVersions=v1

// PaasNSCustomValidator struct is responsible for validating the PaasNS resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
// +kubebuilder:object:generate=false
type PaasNSCustomValidator struct {
}

var _ webhook.CustomValidator = &PaasNSCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paasns, ok := obj.(*PaasNS)
	if !ok {
		return nil, fmt.Errorf("expected a PaasNS object but got %T", obj)
	}

	ctx = setLogComponent(ctx, "paasns_webhook_validate_create")
	logger := log.Ctx(ctx)
	logger.Info().Msgf("Validation for creation of PaasNs %s", paasns.GetName())

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	paasns, ok := newObj.(*PaasNS)
	if !ok {
		return nil, fmt.Errorf("expected a PaasNS object for the newObj but got %T", newObj)
	}
	ctx = setLogComponent(ctx, "paasns_webhook_validate_update")
	logger := log.Ctx(ctx)
	logger.Info().Msgf("Validation for update of PaasNs %s", paasns.GetName())

	// TODO(portly-halicore-76): fill in your validation logic upon object update.

	return nil, nil
}

// TODO(portly-halicore-76): determine whether this can be left out
// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type PaasNS.
func (v *PaasNSCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	paasns, ok := obj.(*PaasNS)
	if !ok {
		return nil, fmt.Errorf("expected a PaasNS object but got %T", obj)
	}
	ctx = setLogComponent(ctx, "paasns_webhook_validate_update")
	logger := log.Ctx(ctx)
	logger.Info().Msgf("Validation for deletion of PaasNs %s", paasns.GetName())

	// TODO(portly-halicore-76): fill in your validation logic upon object deletion.

	return nil, nil
}

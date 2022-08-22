/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kmmv1beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	"github.com/HabanaAI/habana-ai-operator/internal/conditions"
	"github.com/HabanaAI/habana-ai-operator/internal/finalizers"
	"github.com/HabanaAI/habana-ai-operator/internal/metrics"
	nodeMetrics "github.com/HabanaAI/habana-ai-operator/internal/metrics/node"
	"github.com/HabanaAI/habana-ai-operator/internal/module"
	s "github.com/HabanaAI/habana-ai-operator/internal/settings"
)

// Reconciler reconciles a DeviceConfig object
type Reconciler struct {
	client.Client

	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	mr  module.Reconciler
	nmr nodeMetrics.Reconciler

	fu finalizers.Updater
	cu conditions.Updater

	nsv NodeSelectorValidator
}

func NewReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	mr module.Reconciler,
	nmr nodeMetrics.Reconciler,
	fu finalizers.Updater,
	cu conditions.Updater,
	nsv NodeSelectorValidator,
) *Reconciler {
	return &Reconciler{
		Client:   client,
		Scheme:   scheme,
		Recorder: recorder,
		mr:       mr,
		nmr:      nmr,
		fu:       fu,
		cu:       cu,
		nsv:      nsv,
	}
}

//+kubebuilder:rbac:groups=habana.ai,resources=deviceconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=habana.ai,resources=deviceconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=habana.ai,resources=deviceconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups="kmm.sigs.k8s.io",resources=modules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DeviceConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	deviceConfig := &hlaiv1alpha1.DeviceConfig{}
	err := r.Get(ctx, req.NamespacedName, deviceConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			metrics.ReconciliationFailed.WithLabelValues(req.NamespacedName.Name).Set(0)
			logger.Info("DeviceConfig resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get DeviceConfig", "resource", deviceConfig.Name)
		return ctrl.Result{}, err
	}

	if !deviceConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		metrics.ReconciliationFailed.WithLabelValues(deviceConfig.Name).Set(0)

		if r.fu.ContainsDeletionFinalizer(deviceConfig) {
			if err := r.deleteDeviceConfigResources(ctx, deviceConfig); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to delete DeviceConfig resources: %w", err)
			}
			if err := r.fu.RemoveDeletionFinalizer(ctx, deviceConfig); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err := r.nsv.CheckDeviceConfigForConflictingNodeSelector(ctx, deviceConfig); err != nil {
		logger.Error(err, "Failed to validate DeviceConfig", "resource", deviceConfig.Name)
		r.Recorder.Event(
			deviceConfig,
			v1.EventTypeWarning,
			"Error",
			"Conflicting DeviceConfig NodeSelectors found. Please add or update this DeviceConfig's NodeSelector accordingly.",
		)
		metrics.ReconciliationFailed.WithLabelValues(deviceConfig.Name).Set(1)
		return ctrl.Result{}, nil
	}

	if !r.fu.ContainsDeletionFinalizer(deviceConfig) {
		if err := r.fu.AddDeletionFinalizer(ctx, deviceConfig); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.mr.ReconcileModule(ctx, deviceConfig); err != nil {
		if cerr := r.cu.SetConditionsErrored(ctx, deviceConfig, conditions.ReasonModuleFailed, err.Error()); cerr != nil {
			err = fmt.Errorf("%s: %w", err.Error(), cerr)
		}
		metrics.ReconciliationFailed.WithLabelValues(deviceConfig.Name).Set(1)
		return ctrl.Result{}, err
	}

	if err = r.nmr.ReconcileNodeMetrics(ctx, deviceConfig); err != nil {
		if cerr := r.cu.SetConditionsErrored(ctx, deviceConfig, conditions.ReasonNodeMetricsFailed, err.Error()); cerr != nil {
			err = fmt.Errorf("%s: %w", err.Error(), cerr)
		}
		metrics.ReconciliationFailed.WithLabelValues(deviceConfig.Name).Set(1)
		return ctrl.Result{}, err
	}

	metrics.ReconciliationFailed.WithLabelValues(deviceConfig.Name).Set(0)

	r.Recorder.Event(
		deviceConfig,
		v1.EventTypeNormal,
		"Reconciled",
		fmt.Sprintf("Succesfully reconciled DeviceConfig %s/%s", deviceConfig.Namespace, deviceConfig.Name),
	)

	return ctrl.Result{}, r.cu.SetConditionsReady(ctx, deviceConfig, "Reconciled", "All resources have been successfully reconciled")
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := s.Settings.Load()
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named("device-config").
		For(&hlaiv1alpha1.DeviceConfig{}).
		Owns(&kmmv1beta1.Module{}).
		Named("deviceconfig").
		Complete(r)
}

func (r *Reconciler) deleteDeviceConfigResources(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	if err := r.mr.DeleteModule(ctx, cr); err != nil {
		return err
	}

	return nil
}

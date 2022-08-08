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

package module

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	s "github.com/HabanaAI/habana-ai-operator/internal/settings"
	kmmov1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
)

const (
	moduleSuffix = "module"

	driverHabanaServiceAccount = "driver-habana"

	driverHabanaSuffix         = "driver-habana"
	driverHabanaLimitsCpu      = "100m"
	driverHabanaLimitsMemory   = "100Mi"
	driverHabanaRequestsCpu    = "100m"
	driverHabanaRequestsMemory = "50Mi"

	devicePluginSuffix         = "device-plugin"
	devicePluginLimitsCpu      = "200m"
	devicePluginLimitsMemory   = "100Mi"
	devicePluginRequestsCpu    = "100m"
	devicePluginRequestsMemory = "50Mi"
)

//go:generate mockgen -source=module.go -package=module -destination=mock_module.go

type Reconciler interface {
	ReconcileModule(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	SetDesiredModule(m *kmmov1alpha1.Module, cr *hlaiv1alpha1.DeviceConfig) error
	DeleteModule(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
}

type moduleReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewReconciler(c client.Client, s *runtime.Scheme) *moduleReconciler {
	return &moduleReconciler{
		client: c,
		scheme: s,
	}
}

func GetModuleName(cr *hlaiv1alpha1.DeviceConfig) string {
	return fmt.Sprintf("%s-%s", cr.Name, moduleSuffix)
}

func (r *moduleReconciler) ReconcileModule(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	logger := log.FromContext(ctx)

	existingModule := &kmmov1alpha1.Module{}
	err := r.client.Get(ctx, types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      GetModuleName(cr),
	}, existingModule)
	exists := !apierrors.IsNotFound(err)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	m := &kmmov1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetModuleName(cr),
			Namespace: cr.ObjectMeta.Namespace,
		},
	}

	if exists {
		m = existingModule
	}

	res, err := controllerutil.CreateOrPatch(ctx, r.client, m, func() error {
		return r.SetDesiredModule(m, cr)
	})

	if err != nil {
		return fmt.Errorf("could not create or patch Module: %v", err)
	}

	logger.Info("Reconciled Module", "resource", m.Name, "result", res)

	return nil
}

func (r *moduleReconciler) DeleteModule(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	m := &kmmov1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetModuleName(cr),
			Namespace: cr.ObjectMeta.Namespace,
		},
	}

	err := r.client.Delete(ctx, m)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Module %s: %w", m.Name, err)
	}

	return nil
}

func (r *moduleReconciler) SetDesiredModule(m *kmmov1alpha1.Module, cr *hlaiv1alpha1.DeviceConfig) error {
	if m == nil {
		return errors.New("module cannot be nil")
	}

	kernelMappings := []kmmov1alpha1.KernelMapping{
		{
			ContainerImage: fmt.Sprintf("%s:%s-${KERNEL_FULL_VERSION}", cr.Spec.DriverImage, cr.Spec.DriverVersion),
			Regexp:         `^.*\.el\d_?\d?\..*$`,
		},
	}
	driver := r.makeDriverHabanaContainer(cr)
	devicePlugin := r.makeDevicePluginContainer(cr)
	volumes := r.makeAdditionalVolumes(cr)

	m.Spec = kmmov1alpha1.ModuleSpec{
		KernelMappings:    kernelMappings,
		DriverContainer:   driver,
		DevicePlugin:      &devicePlugin,
		AdditionalVolumes: volumes,
	}

	m.Spec.Selector = cr.GetNodeSelector()

	m.Spec.ServiceAccountName = driverHabanaServiceAccount

	if err := ctrl.SetControllerReference(cr, m, r.scheme); err != nil {
		return err
	}

	return nil
}

func (r *moduleReconciler) makeDriverHabanaContainer(cr *hlaiv1alpha1.DeviceConfig) corev1.Container {
	driver := corev1.Container{
		Name: getDriverHabanaName(cr),
	}

	driver.Env = []corev1.EnvVar{
		{
			Name:  "DRIVER_VERSION",
			Value: cr.Spec.DriverVersion,
		},
	}

	driver.ImagePullPolicy = corev1.PullAlways

	privileged := true
	rkmmoUser := int64(0)
	driver.SecurityContext = &corev1.SecurityContext{
		Privileged: &privileged,
		RunAsUser:  &rkmmoUser,
		SELinuxOptions: &corev1.SELinuxOptions{
			Level: "s0",
		},
	}

	driver.Lifecycle = &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/usr/bin/exitpoint",
				},
			},
		},
	}

	driver.ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"sh",
					"-c",
					"lsmod | grep habanalabs",
				},
			},
		},
	}
	driver.LivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"sh",
					"-c",
					"lsmod | grep habanalabs",
				},
			},
		},
	}

	driver.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse(driverHabanaLimitsCpu),
			"memory": resource.MustParse(driverHabanaLimitsMemory),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse(driverHabanaRequestsCpu),
			"memory": resource.MustParse(driverHabanaRequestsMemory),
		},
	}

	mountPropagationBidirectional := corev1.MountPropagationBidirectional
	driver.VolumeMounts = []corev1.VolumeMount{
		{
			Name:             "host-firmware",
			MountPath:        "/var/lib/firmware",
			MountPropagation: &mountPropagationBidirectional,
		},
	}

	return driver
}

func (r *moduleReconciler) makeDevicePluginContainer(cr *hlaiv1alpha1.DeviceConfig) corev1.Container {
	devicePlugin := corev1.Container{
		Name: getDevicePluginName(cr),
	}

	devicePlugin.Env = []corev1.EnvVar{
		{Name: "LD_LIBRARY_PATH", Value: "/usr/lib/habanalabs"},
	}

	devicePlugin.Image = s.Settings.DevicePluginImage
	devicePlugin.ImagePullPolicy = corev1.PullAlways

	devicePlugin.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse(devicePluginLimitsCpu),
			"memory": resource.MustParse(devicePluginLimitsMemory),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse(devicePluginRequestsCpu),
			"memory": resource.MustParse(devicePluginRequestsMemory),
		},
	}

	return devicePlugin
}

func (r *moduleReconciler) makeAdditionalVolumes(cr *hlaiv1alpha1.DeviceConfig) []corev1.Volume {
	hostPathTypeDirectoryOrCreate := corev1.HostPathDirectoryOrCreate
	volumes := []corev1.Volume{
		// We cannot mount the /usr/lib/firmware filesystem from the
		// host to copy the firmware, because it's read-only. Instead
		// we copy it in /var/lib/firmware that is configured as a
		// alternative search path on the node.
		{
			Name: "host-firmware",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/firmware",
					Type: &hostPathTypeDirectoryOrCreate,
				},
			},
		},
	}
	return volumes
}

func getDriverHabanaName(cr *hlaiv1alpha1.DeviceConfig) string {
	return fmt.Sprintf("%s-%s", cr.Name, driverHabanaSuffix)
}

func getDevicePluginName(cr *hlaiv1alpha1.DeviceConfig) string {
	return fmt.Sprintf("%s-%s", cr.Name, devicePluginSuffix)
}

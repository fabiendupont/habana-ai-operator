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

package metrics

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	"github.com/HabanaAI/habana-ai-operator/internal/constants"
	s "github.com/HabanaAI/habana-ai-operator/internal/settings"
)

const (
	nodeMetricsServiceAccount = "node-metrics"
	nodeMetricsSuffix         = "node-metrics"
	nodeMetricsPort           = 41611
	nodeMetricsLimitsCpu      = "1"
	nodeMetricsLimitsMemory   = "200Mi"
	nodeMetricsRequestsCpu    = "100m"
	nodeMetricsRequestsMemory = "200Mi"
)

//go:generate mockgen -source=metrics.go -package=metrics -destination=mock_metrics.go

type Reconciler interface {
	ReconcileNodeMetrics(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	DeleteNodeMetrics(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	ReconcileNodeMetricsDaemonSet(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	SetDesiredNodeMetricsDaemonSet(ds *appsv1.DaemonSet, cr *hlaiv1alpha1.DeviceConfig) error
	DeleteNodeMetricsDaemonSet(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	ReconcileNodeMetricsService(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	SetDesiredNodeMetricsService(s *corev1.Service, cr *hlaiv1alpha1.DeviceConfig) error
	DeleteNodeMetricsService(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
}

type NodeMetricsReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewReconciler(c client.Client, s *runtime.Scheme) *NodeMetricsReconciler {
	return &NodeMetricsReconciler{
		client: c,
		scheme: s,
	}
}

func GetNodeMetricsName(cr *hlaiv1alpha1.DeviceConfig) string {
	return fmt.Sprintf("%s-%s", cr.Name, nodeMetricsSuffix)
}

func (r *NodeMetricsReconciler) ReconcileNodeMetrics(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	err := r.ReconcileNodeMetricsDaemonSet(ctx, cr)
	if err != nil {
		return err
	}

	err = r.ReconcileNodeMetricsService(ctx, cr)
	if err != nil {
		return err
	}

	return setNodeMetricsConditions(r)
}

func (r *NodeMetricsReconciler) ReconcileNodeMetricsDaemonSet(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	logger := log.FromContext(ctx)

	existingDS := &appsv1.DaemonSet{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: GetNodeMetricsName(cr)}, existingDS)
	exists := !apierrors.IsNotFound(err)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNodeMetricsName(cr),
			Namespace: cr.Namespace,
		},
	}

	if exists {
		ds = existingDS
	}

	res, err := controllerutil.CreateOrPatch(ctx, r.client, ds, func() error {
		return r.SetDesiredNodeMetricsDaemonSet(ds, cr)
	})

	if err != nil {
		return fmt.Errorf("could not create or patch DaemonSet: %v", err)
	}

	logger.Info("Reconciled DaemonSet", "resource", ds.Name, "result", res)

	return nil
}

func (r *NodeMetricsReconciler) ReconcileNodeMetricsService(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	logger := log.FromContext(ctx)

	existingService := &corev1.Service{}
	err := r.client.Get(ctx, types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      GetNodeMetricsName(cr),
	}, existingService)
	exists := !apierrors.IsNotFound(err)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNodeMetricsName(cr),
			Namespace: cr.ObjectMeta.Namespace,
		},
	}

	if exists {
		s = existingService
	}

	res, err := controllerutil.CreateOrPatch(ctx, r.client, s, func() error {
		return r.SetDesiredNodeMetricsService(s, cr)
	})

	if err != nil {
		return fmt.Errorf("could not create or patch Service: %v", err)
	}

	logger.Info("Reconciled Service", "resource", s.Name, "result", res)

	return nil
}

func (r *NodeMetricsReconciler) DeleteNodeMetrics(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	err := r.DeleteNodeMetricsDaemonSet(ctx, cr)
	if err != nil {
		return err
	}

	err = r.DeleteNodeMetricsService(ctx, cr)
	if err != nil {
		return err
	}

	return nil
}

func (r *NodeMetricsReconciler) DeleteNodeMetricsDaemonSet(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNodeMetricsName(cr),
			Namespace: cr.Namespace,
		},
	}

	err := r.client.Delete(ctx, ds)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete DaemonSet %s: %w", ds.Name, err)
	}

	return nil
}

func (r *NodeMetricsReconciler) DeleteNodeMetricsService(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNodeMetricsName(cr),
			Namespace: cr.Namespace,
		},
	}

	err := r.client.Delete(ctx, s)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Service %s: %w", s.Name, err)
	}

	return nil
}

func (r *NodeMetricsReconciler) SetDesiredNodeMetricsDaemonSet(ds *appsv1.DaemonSet, cr *hlaiv1alpha1.DeviceConfig) error {
	if ds == nil {
		return errors.New("daemonset cannot be nil")
	}

	labels := labelsForNodeMetricsDaemonSet(cr)

	ds.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}

	ds.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
	}

	hostPathTypeDirectory := corev1.HostPathDirectory
	volumes := []corev1.Volume{
		{
			Name: "pod-resources",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/kubelet/pod-resources",
					Type: &hostPathTypeDirectory,
				},
			},
		},
	}

	containers := []corev1.Container{
		r.makeNodeMetricsContainer(cr),
	}

	nodeSelector := make(map[string]string)
	for k, v := range cr.GetNodeSelector("gaudi") {
		nodeSelector[k] = v
	}

	ds.Spec.Template.Spec = corev1.PodSpec{
		Containers:         containers,
		HostPID:            true,
		NodeSelector:       nodeSelector,
		PriorityClassName:  "system-node-critical",
		ServiceAccountName: nodeMetricsServiceAccount,
		Volumes:            volumes,
	}

	if err := ctrl.SetControllerReference(cr, ds, r.scheme); err != nil {
		return err
	}

	return nil
}

func (r *NodeMetricsReconciler) SetDesiredNodeMetricsService(s *corev1.Service, cr *hlaiv1alpha1.DeviceConfig) error {
	if s == nil {
		return errors.New("service cannot be nil")
	}

	s.ObjectMeta.Labels = labelsForNodeMetricsDaemonSet(cr)
	s.ObjectMeta.Annotations = map[string]string{
		"prometheus.io/scrape": "true",
	}

	s.Spec = corev1.ServiceSpec{
		Selector: labelsForNodeMetricsDaemonSet(cr),
		Ports: []corev1.ServicePort{
			{
				Name:       nodeMetricsSuffix,
				Port:       nodeMetricsPort,
				TargetPort: intstr.FromInt(nodeMetricsPort),
				Protocol:   corev1.ProtocolTCP,
			},
		},
	}

	if err := ctrl.SetControllerReference(cr, s, r.scheme); err != nil {
		return err
	}

	return nil
}

func (r *NodeMetricsReconciler) makeNodeMetricsContainer(cr *hlaiv1alpha1.DeviceConfig) corev1.Container {
	nodeMetrics := corev1.Container{
		Name: nodeMetricsSuffix,
	}

	nodeMetrics.Image = s.Settings.NodeMetricsImage
	nodeMetrics.ImagePullPolicy = corev1.PullAlways

	nodeMetrics.SecurityContext = &corev1.SecurityContext{
		Privileged: pointer.Bool(true),
		RunAsUser:  pointer.Int64(0),
	}

	nodeMetrics.Ports = []corev1.ContainerPort{
		{
			ContainerPort: nodeMetricsPort,
			HostPort:      nodeMetricsPort,
			Name:          nodeMetricsSuffix,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	nodeMetrics.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse(nodeMetricsLimitsCpu),
			"memory": resource.MustParse(nodeMetricsLimitsMemory),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse(nodeMetricsRequestsCpu),
			"memory": resource.MustParse(nodeMetricsRequestsMemory),
		},
	}

	nodeMetrics.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "pod-resources",
			MountPath: "/var/lib/kubelet/pod-resources",
			ReadOnly:  true,
		},
	}

	return nodeMetrics
}

// labelsForNodeMetricsDaemonSet returns the labels for selecting the
// resources belonging to the given DeviceConfig CR name.
func labelsForNodeMetricsDaemonSet(cr *hlaiv1alpha1.DeviceConfig) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      constants.HabanaAIOperatorName,
		"app.kubernetes.io/component": nodeMetricsSuffix,
	}
}

// TODO: NodeMetricsConditions shoud reflect the conditions of the pods
//
//	on the nodes. For that, we should list all the nodes that match the
//	DaemonSet.Spec.Template.Spec.NodeSelector and verify the conditions
//	of the pods running on each of these nodes. We can then set the
//	NodeMetrics* conditions accordingly.
//
// TODO: Define the NodeMetrics* conditions and what they mean.
// setNodeMetricsConditions wraps the condition create/update
func setNodeMetricsConditions(r *NodeMetricsReconciler) (err error) {
	return
}

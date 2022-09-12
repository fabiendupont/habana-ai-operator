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

package labeler

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
	nodeLabelerServiceAccount = "node-labeler"
	nodeLabelerSuffix         = "node-labeler"
	nodeLabelerLimitsCpu      = "1"
	nodeLabelerLimitsMemory   = "200Mi"
	nodeLabelerRequestsCpu    = "100m"
	nodeLabelerRequestsMemory = "200Mi"
)

//go:generate mockgen -source=labeler.go -package=labeler -destination=mock_labeler.go

type Reconciler interface {
	ReconcileNodeLabeler(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	DeleteNodeLabeler(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	ReconcileNodeLabelerDaemonSet(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
	SetDesiredNodeLabelerDaemonSet(ds *appsv1.DaemonSet, cr *hlaiv1alpha1.DeviceConfig) error
	DeleteNodeLabelerDaemonSet(ctx context.Context, dc *hlaiv1alpha1.DeviceConfig) error
}

type NodeLabelerReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewReconciler(c client.Client, s *runtime.Scheme) *NodeLabelerReconciler {
	return &NodeLabelerReconciler{
		client: c,
		scheme: s,
	}
}

func GetNodeLabelerName(cr *hlaiv1alpha1.DeviceConfig) string {
	return fmt.Sprintf("%s-%s", cr.Name, nodeLabelerSuffix)
}

func (r *NodeLabelerReconciler) ReconcileNodeLabeler(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	err := r.ReconcileNodeLabelerDaemonSet(ctx, cr)
	if err != nil {
		return err
	}

	return setNodeLabelerConditions(r)
}

func (r *NodeLabelerReconciler) ReconcileNodeLabelerDaemonSet(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	logger := log.FromContext(ctx)

	existingDS := &appsv1.DaemonSet{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: GetNodeLabelerName(cr)}, existingDS)
	exists := !apierrors.IsNotFound(err)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNodeLabelerName(cr),
			Namespace: cr.Namespace,
		},
	}

	if exists {
		ds = existingDS
	}

	res, err := controllerutil.CreateOrPatch(ctx, r.client, ds, func() error {
		return r.SetDesiredNodeLabelerDaemonSet(ds, cr)
	})

	if err != nil {
		return fmt.Errorf("could not create or patch DaemonSet: %v", err)
	}

	logger.Info("Reconciled DaemonSet", "resource", ds.Name, "result", res)

	return nil
}

func (r *NodeLabelerReconciler) DeleteNodeLabeler(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	err := r.DeleteNodeLabelerDaemonSet(ctx, cr)
	if err != nil {
		return err
	}

	return nil
}

func (r *NodeLabelerReconciler) DeleteNodeLabelerDaemonSet(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNodeLabelerName(cr),
			Namespace: cr.Namespace,
		},
	}

	err := r.client.Delete(ctx, ds)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete DaemonSet %s: %w", ds.Name, err)
	}

	return nil+
}

func (r *NodeLabelerReconciler) SetDesiredNodeLabelerDaemonSet(ds *appsv1.DaemonSet, cr *hlaiv1alpha1.DeviceConfig) error {
	if ds == nil {
		return errors.New("daemonset cannot be nil")
	}

	labels := labelsForNodeLabelerDaemonSet(cr)

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
					Path: "/etc/kubernetes/node-feature-discovery/features.d",
					Type: &hostPathTypeDirectory,
				},
			},
		},
	}

	containers := []corev1.Container{
		r.makeNodeLabelerContainer(cr),
	}

	nodeSelector := make(map[string]string)
	for k, v := range cr.GetNodeSelector() {
		nodeSelector[k] = v
	}

	ds.Spec.Template.Spec = corev1.PodSpec{
		Containers:         containers,
		HostPID:            true,
		NodeSelector:       nodeSelector,
		PriorityClassName:  "system-node-critical",
		ServiceAccountName: nodeLabelerServiceAccount,
		Volumes:            volumes,
	}

	if err := ctrl.SetControllerReference(cr, ds, r.scheme); err != nil {
		return err
	}

	return nil
}

func (r *NodeLabelerReconciler) makeNodeLabelerContainer(cr *hlaiv1alpha1.DeviceConfig) corev1.Container {
	nodeLabeler := corev1.Container{
		Name: nodeLabelerSuffix,
	}

	nodeLabeler.Image = s.Settings.NodeLabelerImage
	nodeLabeler.ImagePullPolicy = corev1.PullAlways

	nodeLabeler.SecurityContext = &corev1.SecurityContext{
		Privileged: pointer.Bool(true),
		RunAsUser:  pointer.Int64(0),
	}

	nodeLabeler.Ports = []corev1.ContainerPort{
		{
			ContainerPort: nodeLabelerPort,
			HostPort:      nodeLabelerPort,
			Name:          nodeLabelerSuffix,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	nodeLabeler.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse(nodeLabelerLimitsCpu),
			"memory": resource.MustParse(nodeLabelerLimitsMemory),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse(nodeLabelerRequestsCpu),
			"memory": resource.MustParse(nodeLabelerRequestsMemory),
		},
	}

	nodeLabeler.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "pod-resources",
			MountPath: "/etc/kubernetes/node-feature-discovery/features.d",
			ReadOnly:  false,
		},
	}

	return nodeLabeler
}

// labelsForNodeLabelerDaemonSet returns the labels for selecting the
// resources belonging to the given DeviceConfig CR name.
func labelsForNodeLabelerDaemonSet(cr *hlaiv1alpha1.DeviceConfig) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      constants.HabanaAIOperatorName,
		"app.kubernetes.io/component": nodeLabelerSuffix,
	}
}

// TODO: NodeLabelerConditions shoud reflect the conditions of the pods
//       on the nodes. For that, we should list all the nodes that match the
//       DaemonSet.Spec.Template.Spec.NodeSelector and verify the conditions
//       of the pods running on each of these nodes. We can then set the
//       NodeLabeler* conditions accordingly.
// TODO: Define the NodeLabeler* conditions and what they mean.
// setNodeLabelerConditions wraps the condition create/update
func setNodeLabelerConditions(r *NodeLabelerReconciler) (err error) {
	return
}

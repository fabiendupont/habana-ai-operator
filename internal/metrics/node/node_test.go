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

package node

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	s "github.com/HabanaAI/habana-ai-operator/internal/settings"
)

const (
	testLabelKey = "label"

	testLabelValue = "test"

	testNFDLabelKey = "feature.node.kubernetes.io/pci-1da3.present"
)

func TestNodeMetricsReconciler_ReconcileNodeMetricsDaemonSet(t *testing.T) {
	dc := makeTestDeviceConfig()
	r := makeTestReconciler(t, dc)
	ds := &appsv1.DaemonSet{}

	// check that resource is created
	assert.NoError(t, r.ReconcileNodeMetricsDaemonSet(context.TODO(), dc))
	assert.NoError(t, r.client.Get(context.TODO(), types.NamespacedName{Name: GetNodeMetricsName(dc)}, ds))

	// check that resource is updated

	// test that the nodeSelector is applied correctly
	ns := map[string]string{testLabelKey: testLabelValue}
	dc.Spec.NodeSelector = ns
	assert.NoError(t, r.ReconcileNodeMetricsDaemonSet(context.TODO(), dc))
	assert.NoError(t, r.client.Get(context.TODO(), types.NamespacedName{Name: ds.Name}, ds))

	// check that it contains the specified selector
	v, contains := ds.Spec.Template.Spec.NodeSelector[testLabelKey]
	assert.True(t, contains)
	assert.Equal(t, testLabelValue, v)

	// check that it does not contain the default NFD selector
	assert.NotContains(t, ds.Spec.Template.Spec.NodeSelector, testNFDLabelKey)
}

func TestNodeMetricsReconciler_DeleteNodeMetricsDaemonSet(t *testing.T) {
	dc := makeTestDeviceConfig()
	r := makeTestReconciler(t, dc)
	ds := &appsv1.DaemonSet{}

	// test that resource is created
	assert.NoError(t, r.ReconcileNodeMetricsDaemonSet(context.TODO(), dc))
	assert.NoError(t, r.client.Get(context.TODO(), types.NamespacedName{Name: GetNodeMetricsName(dc)}, ds))

	// test that resource is deleted
	assert.NoError(t, r.DeleteNodeMetricsDaemonSet(context.TODO(), dc))

	nodeMetricsDaemonSet := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: ds.Name}, nodeMetricsDaemonSet)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected not found error, got %#v\n", err)
	}
}

func TestNodeMetricsReconciler_ReconcileNodeMetricsService(t *testing.T) {
	dc := makeTestDeviceConfig()
	r := makeTestReconciler(t, dc)
	s := &corev1.Service{}

	assert.NoError(t, r.ReconcileNodeMetricsService(context.TODO(), dc))
	assert.NoError(t, r.client.Get(context.TODO(), types.NamespacedName{Name: GetNodeMetricsName(dc)}, s))
	assert.Equal(t, GetNodeMetricsName(dc), s.Name)
}

func TestNodeMetricsReconciler_DeleteNodeMetricsService(t *testing.T) {
	dc := makeTestDeviceConfig()
	r := makeTestReconciler(t, dc)
	s := &corev1.Service{}

	// test that resource is created
	assert.NoError(t, r.ReconcileNodeMetricsService(context.TODO(), dc))
	assert.NoError(t, r.client.Get(context.TODO(), types.NamespacedName{Name: GetNodeMetricsName(dc)}, s))

	// test that resource is deleted
	assert.NoError(t, r.DeleteNodeMetricsService(context.TODO(), dc))

	nodeMetricsService := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: s.Name}, nodeMetricsService)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected not found error, got %#v\n", err)
	}
}

func TestNodeMetricsReconciler_SetDesiredNodeMetricsDaemonSet(t *testing.T) {
	dc := makeTestDeviceConfig()
	r := makeTestReconciler(t, dc)
	ds := &appsv1.DaemonSet{}

	err := r.SetDesiredNodeMetricsDaemonSet(ds, dc)
	assert.NoError(t, err)

	// test that the DaemonSet's service account it the predefined one
	assert.Equal(t, nodeMetricsServiceAccount, ds.Spec.Template.Spec.ServiceAccountName)

	// test that the DaemonSet's HostPID is enabled
	assert.True(t, ds.Spec.Template.Spec.HostPID)

	// test that nodeMetrics container is specified
	assert.Len(t, ds.Spec.Template.Spec.Containers, 1)

	assert.Len(t, ds.Spec.Template.Spec.Volumes, 1)

	// Without a NodeSelector defined in the CR, check if the default NFD label
	// is included in the NodeSelector.
	assert.Contains(t, ds.Spec.Template.Spec.NodeSelector, testNFDLabelKey)
}

func TestNodeMetricsReconciler_SetDesiredNodeMetricsService(t *testing.T) {
	dc := makeTestDeviceConfig()
	r := makeTestReconciler(t, dc)
	service := &corev1.Service{}

	err := r.SetDesiredNodeMetricsService(service, dc)
	assert.NoError(t, err)

	assert.NotNil(t, service.Spec.Ports)
	assert.Len(t, service.Spec.Ports, 1)
	assert.EqualValues(t, nodeMetricsPort, service.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(nodeMetricsPort), service.Spec.Ports[0].TargetPort)
	assert.Equal(t, corev1.ProtocolTCP, service.Spec.Ports[0].Protocol)
	assert.Contains(t, service.Annotations, "prometheus.io/scrape")
}

func TestNodeMetricsReconciler_makeNodeMetricsContainer(t *testing.T) {
	dc := makeTestDeviceConfig()
	r := makeTestReconciler(t, dc)

	nodeMetrics := r.makeNodeMetricsContainer(dc)

	assert.Equal(t, nodeMetricsSuffix, nodeMetrics.Name)
	assert.Equal(t, s.Settings.NodeMetricsImage, nodeMetrics.Image)
	assert.Equal(t, corev1.PullAlways, nodeMetrics.ImagePullPolicy)
	assert.True(t, *nodeMetrics.SecurityContext.Privileged)
	assert.NotNil(t, nodeMetrics.Ports)
	assert.NotNil(t, nodeMetrics.Resources)
	assert.Len(t, nodeMetrics.VolumeMounts, 1)
}

func makeTestDeviceConfig() *hlaiv1alpha1.DeviceConfig {
	c := &hlaiv1alpha1.DeviceConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "a-device-config",
		},
		Spec: hlaiv1alpha1.DeviceConfigSpec{
			DriverImage:   "",
			DriverVersion: "",
		},
	}
	return c
}

func makeTestReconciler(t *testing.T, objs ...runtime.Object) *NodeMetricsReconciler {
	s := scheme.Scheme
	assert.NoError(t, hlaiv1alpha1.AddToScheme(s))

	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
	return &NodeMetricsReconciler{
		client: cl,
		scheme: s,
	}
}

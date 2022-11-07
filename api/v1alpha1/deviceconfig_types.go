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

package v1alpha1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeviceConfigDeletionFinalizer = "device-config-deletion-finalizer"

	HabanaPCIVendorID = "1da3"
)

// DeviceConfigSpec defines the desired state of DeviceConfig
type DeviceConfigSpec struct {
	//+kubebuilder:validation:Required
	// DriverImage is the Habana driver image to use
	DriverImage string `json:"driverImage"`
	//+kubebuilder:validation:Required
	// DriverVersion is the Habana driver version deployed
	DriverVersion string `json:"driverVersion"`
	//+kubebuilder:validation:Optional
	// NodeSelector specifies a selector for the DeviceConfig
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	//+kubebuilder:validation:Optional
	// GaudiNodeSelector specifies a selector for Gaudi nodes
	GaudiNodeSelector map[string]string `json:"gaudiNodeSelector,omitempty"`
}

// DeviceConfigStatus defines the observed state of DeviceConfig
type DeviceConfigStatus struct {
	// Conditions is a list of conditions representing the DeviceConfig's current state.
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeviceConfig is the Schema for the deviceconfigs API
type DeviceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceConfigSpec   `json:"spec,omitempty"`
	Status DeviceConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeviceConfigList contains a list of DeviceConfig
type DeviceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeviceConfig{}, &DeviceConfigList{})
}

func (dc *DeviceConfig) GetNodeSelector() map[string]string {
	ns := dc.Spec.NodeSelector
	if ns == nil {
		ns = make(map[string]string, 0)
		// If no DeviceConfig.NodeSelector is specified, let's try adding NFD labels, otherwise
		// the daemonset would be deployed on every schedulable node.
		ns[fmt.Sprintf("feature.node.kubernetes.io/pci-%s.present", HabanaPCIVendorID)] = "true"
	}
	return ns
}

func (dc *DeviceConfig) GetGaudiNodeSelector() map[string]string {
	ns := dc.Spec.GaudiNodeSelector
	if ns == nil {
		ns = make(map[string]string, 0)
		// If no DeviceConfig.NodeSelector is specified, let's try adding NFD labels, otherwise
		// the daemonset would be deployed on every schedulable node.
		deviceType := "gaudi"
		ns[fmt.Sprintf("habana.ai/hpu.%s.present", deviceType)] = "true"
	}
	return ns
}

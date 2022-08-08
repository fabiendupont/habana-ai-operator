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
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
)

//go:generate mockgen -source=nodeselector.go -package=controllers -destination=mock_nodeselector.go

type NodeSelectorValidator interface {
	CheckDeviceConfigForConflictingNodeSelector(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error
}

type nodeSelectorValidator struct {
	client client.Client
}

func NewNodeSelectorValidator(c client.Client) *nodeSelectorValidator {
	return &nodeSelectorValidator{client: c}
}

func (nsv *nodeSelectorValidator) CheckDeviceConfigForConflictingNodeSelector(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	dcs := &hlaiv1alpha1.DeviceConfigList{}
	err := nsv.client.List(ctx, dcs)
	if err != nil {
		return err
	}

	names := []string{}
	for _, dc := range dcs.Items {
		nodeList, err := nsv.getDeviceConfigSelectedNodes(ctx, &dc)
		if err != nil {
			return err
		}

		for _, n := range nodeList.Items {
			names = append(names, n.Name)
		}
	}

	if containsDuplicates(names) {
		return fmt.Errorf("conflicting DeviceConfig NodeSelectors found for resource: %s", cr.Name)
	}

	return nil
}

func (nsv *nodeSelectorValidator) getDeviceConfigSelectedNodes(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) (*v1.NodeList, error) {
	nodeList := &v1.NodeList{}

	if cr.Spec.NodeSelector == nil {
		cr.Spec.NodeSelector = cr.GetNodeSelector()
	}

	selector := labels.Set(cr.Spec.NodeSelector).AsSelector()

	opts := []client.ListOption{
		client.MatchingLabelsSelector{Selector: selector},
	}
	err := nsv.client.List(ctx, nodeList, opts...)

	return nodeList, err
}

func containsDuplicates(arr []string) bool {
	visited := make(map[string]bool, 0)
	for _, e := range arr {
		if _, exists := visited[e]; exists {
			return true
		}
		visited[e] = true
	}
	return false
}

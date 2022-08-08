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

package finalizers

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
)

//go:generate mockgen -source=finalizers.go -package=finalizers -destination=mock_finalizers.go

type Updater interface {
	AddDeletionFinalizer(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error
	RemoveDeletionFinalizer(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error
	ContainsDeletionFinalizer(cr *hlaiv1alpha1.DeviceConfig) bool
}

type updater struct {
	statusWriter client.StatusWriter
}

func NewUpdater(sw client.StatusWriter) Updater {
	return &updater{statusWriter: sw}
}

func (u *updater) AddDeletionFinalizer(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	controllerutil.AddFinalizer(cr, hlaiv1alpha1.DeviceConfigDeletionFinalizer)
	if err := u.statusWriter.Update(ctx, cr); err != nil {
		return fmt.Errorf("failed to add deletion finalizer for %s: %w", cr.Name, err)
	}
	return nil
}

func (u *updater) RemoveDeletionFinalizer(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig) error {
	controllerutil.RemoveFinalizer(cr, hlaiv1alpha1.DeviceConfigDeletionFinalizer)
	if err := u.statusWriter.Update(ctx, cr); err != nil {
		return fmt.Errorf("failed to remove deletion finalizer for %s: %w", cr.Name, err)
	}
	return nil
}

func (u *updater) ContainsDeletionFinalizer(cr *hlaiv1alpha1.DeviceConfig) bool {
	return controllerutil.ContainsFinalizer(cr, hlaiv1alpha1.DeviceConfigDeletionFinalizer)
}

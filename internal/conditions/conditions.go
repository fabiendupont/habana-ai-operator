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

package conditions

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
)

const (
	Ready = "Ready"

	Errored = "Errored"

	ReasonModuleFailed = "ModuleFailed"

	ReasonConflictingNodeSelector = "ConflictingNodeSelector"
)

//go:generate mockgen -source=conditions.go -package=conditions -destination=mock_conditions.go

type Updater interface {
	SetConditionsReady(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig, reason, message string) error
	SetConditionsErrored(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig, reason, message string) error
}

type updater struct {
	statusWriter client.StatusWriter
}

func NewUpdater(sw client.StatusWriter) Updater {
	return &updater{statusWriter: sw}
}

func (u *updater) SetConditionsReady(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig, reason, message string) error {
	meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    Ready,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})

	meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:   Errored,
		Status: metav1.ConditionFalse,
		Reason: Ready,
	})

	return u.statusWriter.Update(ctx, cr)
}

func (u *updater) SetConditionsErrored(ctx context.Context, cr *hlaiv1alpha1.DeviceConfig, reason, message string) error {
	meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:   Ready,
		Status: metav1.ConditionFalse,
		Reason: Errored,
	})

	meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    Errored,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})

	return u.statusWriter.Update(ctx, cr)
}

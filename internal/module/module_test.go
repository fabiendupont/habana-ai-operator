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

	gomock "github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	mockClient "github.com/HabanaAI/habana-ai-operator/internal/client"
	kmmov1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
)

const (
	testDriverImage   = "driver"
	testDriverVersion = "test"

	testLabelKey   = "label"
	testLabelValue = "test"
)

var _ = Describe("ModuleReconciler", func() {
	var (
		dc  *hlaiv1alpha1.DeviceConfig
		r   *moduleReconciler
		c   *mockClient.MockClient
		ctx context.Context
	)

	BeforeEach(func() {
		dc = &hlaiv1alpha1.DeviceConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "a-device-config",
				Namespace: "a-namespace",
			},
		}
		c = mockClient.NewMockClient(gomock.NewController(GinkgoT()))

		s := scheme.Scheme
		Expect(hlaiv1alpha1.AddToScheme(s)).ToNot(HaveOccurred())
		Expect(kmmov1alpha1.AddToScheme(s)).ToNot(HaveOccurred())
		r = NewReconciler(c, s)

		ctx = context.TODO()
	})

	Describe("ReconcileModule", func() {
		Context("with no client Get error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().
						Get(ctx, gomock.Any(), gomock.Any()).
						Return(apierrors.NewNotFound(schema.GroupResource{Resource: "modules"}, GetModuleName(dc))).
						AnyTimes(),
				)
			})

			Context("with no client Create error", func() {
				BeforeEach(func() {
					gomock.InOrder(
						c.EXPECT().Create(ctx, gomock.Any()).Return(nil),
					)
				})
				It("should not return an error", func() {
					Expect(r.ReconcileModule(ctx, dc)).ToNot(HaveOccurred())
				})
			})

			Context("with client Create error", func() {
				BeforeEach(func() {
					gomock.InOrder(
						c.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("some-error")),
					)
				})
				It("should return an error", func() {
					Expect(r.ReconcileModule(ctx, dc)).To(HaveOccurred())
				})
			})
		})

		Context("with client Get error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(errors.New("some-other-that-not-found-error")),
				)
			})

			It("should return an error", func() {
				Expect(r.ReconcileModule(ctx, dc)).To(HaveOccurred())
			})
		})
	})

	Describe("DeleteModule", func() {
		Context("without a client Delete error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, gomock.Any()).Return(nil),
				)
			})

			It("should not return an error", func() {
				Expect(r.DeleteModule(ctx, dc)).ToNot(HaveOccurred())
			})
		})

		Context("with a NotFound client Delete error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().
						Delete(ctx, gomock.Any()).
						Return(apierrors.NewNotFound(schema.GroupResource{Resource: "modules"}, GetModuleName(dc))),
				)
			})

			It("should not return an error", func() {
				Expect(r.DeleteModule(ctx, dc)).ToNot(HaveOccurred())
			})
		})

		Context("with a generic client Delete error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, gomock.Any()).Return(errors.New("some-error")),
				)
			})

			It("should return an error", func() {
				Expect(r.DeleteModule(ctx, dc)).To(HaveOccurred())
			})
		})
	})

	Describe("SetDesiredModule", func() {
		var (
			m *kmmov1alpha1.Module
		)

		Context("with a nil Module as input", func() {
			BeforeEach(func() {
				m = nil
			})

			It("should return a module cannot be nil error", func() {
				err := r.SetDesiredModule(m, dc)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("module cannot be nil"))
			})
		})

		Context("with a non-nil Module as input", func() {
			BeforeEach(func() {
				dc.Spec.NodeSelector = map[string]string{testLabelKey: testLabelValue}
				dc.Spec.DriverImage = testDriverImage
				dc.Spec.DriverVersion = testDriverVersion

				m = &kmmov1alpha1.Module{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a-name",
						Namespace: "a-namespace",
					},
				}

				err := r.SetDesiredModule(m, dc)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("it returns a Module which", func() {
				It("should contain the correct node selector", func() {
					Expect(m.Spec.Selector).ToNot(BeNil())

					v, contains := m.Spec.Selector[testLabelKey]
					Expect(contains).To(BeTrue())
					Expect(v).To(Equal(testLabelValue))
				})

				It("should contain the correct KernelMappings", func() {
					Expect(m.Spec.KernelMappings).ToNot(BeNil())
					Expect(m.Spec.KernelMappings).To(HaveLen(1))

					expectedImage := fmt.Sprintf("%s:%s-${KERNEL_FULL_VERSION}", testDriverImage, testDriverVersion)
					Expect(m.Spec.KernelMappings[0].ContainerImage).To(Equal(expectedImage))
				})

				It("should have the correct DriverContainer", func() {
					Expect(m.Spec.DriverContainer).ToNot(BeNil())
					Expect(m.Spec.DriverContainer.Lifecycle).ToNot(BeNil())
					Expect(m.Spec.DriverContainer.Lifecycle.PreStop).ToNot(BeNil())
					Expect(m.Spec.DriverContainer.ReadinessProbe).ToNot(BeNil())
					Expect(m.Spec.DriverContainer.LivenessProbe).ToNot(BeNil())
					Expect(m.Spec.DriverContainer.Env).ToNot(BeNil())
					Expect(m.Spec.DriverContainer.ImagePullPolicy).To(Equal(corev1.PullAlways))
					Expect(m.Spec.DriverContainer.VolumeMounts).To(HaveLen(1))
					Expect(m.Spec.DriverContainer.VolumeMounts[0].Name).To(Equal("host-firmware"))
					Expect(*m.Spec.DriverContainer.SecurityContext.Privileged).To(BeTrue())
					Expect(*m.Spec.DriverContainer.SecurityContext.RunAsUser).To(BeZero())
				})

				It("should have a correct DevicePlugin", func() {
					Expect(m.Spec.DevicePlugin).ToNot(BeNil())
					Expect(m.Spec.DevicePlugin.Resources).ToNot(BeNil())
					Expect(m.Spec.DevicePlugin.Env).ToNot(BeNil())
					Expect(m.Spec.DevicePlugin.ImagePullPolicy).To(Equal(corev1.PullAlways))
				})

				It("should have the correct SA", func() {
					Expect(m.Spec.ServiceAccountName).To(Equal(driverHabanaServiceAccount))
				})

				It("should have the correct AdditionalVolumes", func() {
					Expect(m.Spec.AdditionalVolumes).ToNot(BeNil())
					Expect(m.Spec.AdditionalVolumes).To(HaveLen(1))
					Expect(m.Spec.AdditionalVolumes[0].Name).To(Equal("host-firmware"))
				})
			})
		})
	})
})

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

	gomock "github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	"github.com/HabanaAI/habana-ai-operator/internal/client"
	s "github.com/HabanaAI/habana-ai-operator/internal/settings"
)

const (
	testLabelKey = "habana.ai/hpu.gaudi.present"
	testLabelValue = "true"
)

var _ = Describe("NodeLabelerReconciler", func() {
	var (
		dc  *hlaiv1alpha1.DeviceConfig
		r   *NodeLabelerReconciler
		c   *client.MockClient
		ctx context.Context
	)

	BeforeEach(func() {
		dc = &hlaiv1alpha1.DeviceConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "a-device-config",
				Namespace: "a-namespace",
			},
		}
		c = client.NewMockClient(gomock.NewController(GinkgoT()))

		s := scheme.Scheme
		Expect(hlaiv1alpha1.AddToScheme(s)).ToNot(HaveOccurred())

		r = NewReconciler(c, s)

		ctx = context.TODO()
	})

	Describe("ReconcileNodeLabeler", func() {
	})

	Describe("ReconcileNodeLabelerDaemonSet", func() {
		Context("with no client Get error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().
						Get(ctx, gomock.Any(), gomock.Any()).
						Return(apierrors.NewNotFound(schema.GroupResource{Resource: "daemonsets"}, GetNodeLabelerName(dc))).
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
					Expect(r.ReconcileNodeLabelerDaemonSet(ctx, dc)).ToNot(HaveOccurred())
				})
			})

			Context("with client Create error", func() {
				BeforeEach(func() {
					gomock.InOrder(
						c.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("some-error")),
					)
				})

				It("should return an error", func() {
					Expect(r.ReconcileNodeLabelerDaemonSet(ctx, dc)).To(HaveOccurred())
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
				Expect(r.ReconcileNodeLabelerDaemonSet(ctx, dc)).To(HaveOccurred())
			})
		})
	})

	Describe("DeleteNodeLabelerDaemonSet", func() {
		Context("without a client Delete error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, gomock.Any()).Return(nil),
				)
			})

			It("should not return an error", func() {
				Expect(r.DeleteNodeLabelerDaemonSet(ctx, dc)).ToNot(HaveOccurred())
			})
		})

		Context("with a NotFound client Delete error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().
						Delete(ctx, gomock.Any()).
						Return(apierrors.NewNotFound(schema.GroupResource{Resource: "daemonsets"}, GetNodeLabelerName(dc))),
				)
			})

			It("should not return an error", func() {
				Expect(r.DeleteNodeLabelerDaemonSet(ctx, dc)).ToNot(HaveOccurred())
			})
		})

		Context("with a generic client Delete error", func() {
			BeforeEach(func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, gomock.Any()).Return(errors.New("some-error")),
				)
			})

			It("should return an error", func() {
				Expect(r.DeleteNodeLabelerDaemonSet(ctx, dc)).To(HaveOccurred())
			})
		})
	})

	Describe("SetDesiredNodeLabelerDaemonSet", func() {
		var (
			ds *appsv1.DaemonSet
		)

		Context("with a nil DaemonSet as input", func() {
			BeforeEach(func() {
				ds = nil
			})

			It("should return a DaemonSet cannot be nil error", func() {
				err := r.SetDesiredNodeLabelerDaemonSet(ds, dc)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("daemonset cannot be nil"))
			})
		})

		Context("with a non-nil DaemonSet as input", func() {
			BeforeEach(func() {
				dc.Spec.NodeSelector = map[string]string{testLabelKey: testLabelValue}

				ds = &appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a-name",
						Namespace: "a-namespace",
					},
				}

				err := r.SetDesiredNodeLabelerDaemonSet(ds, dc)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("it returns a DaemonSet which", func() {
				It("should contain the correct node selector", func() {
					Expect(ds.Spec.Template.Spec.NodeSelector).ToNot(BeNil())

					v, contains := ds.Spec.Template.Spec.NodeSelector[testLabelKey]
					Expect(contains).To(BeTrue())
					Expect(v).To(Equal(testLabelValue))
				})

				It("should have the HostPID enabled", func() {
					Expect(ds.Spec.Template.Spec.HostPID).To(BeTrue())
				})

				It("should have the correct ServiceAccountName", func() {
					Expect(ds.Spec.Template.Spec.ServiceAccountName).To(Equal(nodeLabelerServiceAccount))
				})

				It("should have one container", func() {
					Expect(ds.Spec.Template.Spec.Containers).To(HaveLen(1))
				})

				Context("contains the nodeLabeler container, which", func() {
					var (
						nodeLabeler corev1.Container
					)

					BeforeEach(func() {
						nodeLabeler = ds.Spec.Template.Spec.Containers[0]
					})

					It("should have the correct name", func() {
						Expect(nodeLabeler.Name).To(Equal(nodeLabelerSuffix))
					})

					It("should have the correct image", func() {
						Expect(nodeLabeler.Image).To(Equal(s.Settings.NodeLabelerImage))
					})

					It("should have the image pull policy always", func() {
						Expect(nodeLabeler.ImagePullPolicy).To(Equal(corev1.PullAlways))
					})

					It("should have the privileged SecurityContext", func() {
						Expect(*nodeLabeler.SecurityContext.Privileged).To(BeTrue())
					})

					It("should run as user root (id: 0)", func() {
						Expect(*nodeLabeler.SecurityContext.RunAsUser).To(Equal(int64(0)))
					})

					It("should have resources", func() {
						Expect(nodeLabeler.Resources).ToNot(BeNil())
					})

					It("should have one volume mount ", func() {
						Expect(nodeLabeler.VolumeMounts).To(HaveLen(1))
					})
				})

				It("should have one volume", func() {
					Expect(ds.Spec.Template.Spec.Volumes).To(HaveLen(1))
				})
			})
		})
	})
})

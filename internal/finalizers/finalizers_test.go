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
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	"github.com/HabanaAI/habana-ai-operator/internal/client"
)

var _ = Describe("FinalizersUpdater", func() {
	var (
		dc *hlaiv1alpha1.DeviceConfig
		c  *client.MockClient
		u  Updater
	)

	BeforeEach(func() {
		dc = &hlaiv1alpha1.DeviceConfig{ObjectMeta: metav1.ObjectMeta{Name: "a-device-config"}}
		c = client.NewMockClient(gomock.NewController(GinkgoT()))
		u = NewUpdater(c)
	})

	Describe("AddDeletionFinalizer", func() {
		Context("with a successful update", func() {
			BeforeEach(func() {
				c.EXPECT().Update(context.TODO(), dc)
			})

			It("should add the deletion finalizer to the DeviceConfig", func() {
				err := u.AddDeletionFinalizer(context.TODO(), dc)
				Expect(err).ToNot(HaveOccurred())
				Expect(dc.Finalizers).To(ContainElement(hlaiv1alpha1.DeviceConfigDeletionFinalizer))
			})
		})

		Context("with a failed update", func() {
			BeforeEach(func() {
				c.EXPECT().Update(context.TODO(), dc).Return(errors.New("some error"))
			})

			It("should return an error", func() {
				err := u.AddDeletionFinalizer(context.TODO(), dc)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("RemoveDeletionFinalizer", func() {
		BeforeEach(func() {
			dc.SetFinalizers([]string{hlaiv1alpha1.DeviceConfigDeletionFinalizer})
		})

		Context("with a successful update", func() {
			BeforeEach(func() {
				c.EXPECT().Update(context.TODO(), dc)
			})

			It("should remove the finalizer from the DeviceConfig", func() {
				err := u.RemoveDeletionFinalizer(context.TODO(), dc)
				Expect(err).ToNot(HaveOccurred())
				Expect(dc.Finalizers).ToNot(ContainElement(hlaiv1alpha1.DeviceConfigDeletionFinalizer))
			})
		})

		Context("with a failed update", func() {
			BeforeEach(func() {
				c.EXPECT().Update(context.TODO(), dc).Return(errors.New("some error"))
			})

			It("should return an error", func() {
				err := u.RemoveDeletionFinalizer(context.TODO(), dc)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ContainsDeletionFinalizer", func() {
		Context("for a DeviceConfig with a deletion finalizer", func() {
			BeforeEach(func() {
				dc.SetFinalizers([]string{hlaiv1alpha1.DeviceConfigDeletionFinalizer})
			})

			It("should return true", func() {
				Expect(u.ContainsDeletionFinalizer(dc)).To(BeTrue())
			})
		})

		Context("for a DeviceConfig without a deletion finalizer", func() {
			It("should return false", func() {
				Expect(u.ContainsDeletionFinalizer(dc)).To(BeFalse())
			})
		})
	})
})

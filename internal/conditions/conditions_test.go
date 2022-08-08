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
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	mockClient "github.com/HabanaAI/habana-ai-operator/internal/client"
)

var _ = Describe("ConditionsUpdater", func() {
	var (
		dc *hlaiv1alpha1.DeviceConfig
		c  *mockClient.MockClient
		u  Updater
	)

	BeforeEach(func() {
		dc = &hlaiv1alpha1.DeviceConfig{ObjectMeta: metav1.ObjectMeta{Name: "a-device-config"}}
		c = mockClient.NewMockClient(gomock.NewController(GinkgoT()))
		u = NewUpdater(c)
	})

	Describe("SetConditionsReady", func() {
		Context("with successful status update", func() {
			BeforeEach(func() {
				c.EXPECT().Update(context.TODO(), dc)

				err := u.SetConditionsReady(context.TODO(), dc, "test reason", "test message")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should have set 2 conditions in the DeviceConfig .status.conditions", func() {
				Expect(dc.Status.Conditions).To(HaveLen(2))
			})

			It("should have set the Ready condition as true", func() {
				ready := dc.Status.Conditions[0]

				Expect(ready.Type).To(Equal("Ready"))
				Expect(ready.Status).To(Equal(metav1.ConditionTrue))
				Expect(ready.Reason).To(Equal("test reason"))
				Expect(ready.Message).To(Equal("test message"))
			})

			It("should have set the Errored condition as false", func() {
				errored := dc.Status.Conditions[1]

				Expect(errored.Type).To(Equal("Errored"))
				Expect(errored.Status).To(Equal(metav1.ConditionFalse))
				Expect(errored.Reason).To(Equal("Ready"))
			})
		})

		Context("with failed status update", func() {
			BeforeEach(func() {
				c.EXPECT().Update(context.TODO(), dc).Return(errors.New("some error"))
			})

			It("should return an error", func() {
				err := u.SetConditionsReady(context.TODO(), dc, "test reason", "test message")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("SetConditionsErrored", func() {
		Context("with successful status update", func() {
			BeforeEach(func() {
				c.EXPECT().Update(context.TODO(), dc)

				err := u.SetConditionsErrored(context.TODO(), dc, "test reason", "test message")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should have set 2 conditions in the DeviceConfig .status.conditions", func() {
				Expect(dc.Status.Conditions).To(HaveLen(2))
			})

			It("should have set the Ready condition as false", func() {
				ready := dc.Status.Conditions[0]

				Expect(ready.Type).To(Equal("Ready"))
				Expect(ready.Status).To(Equal(metav1.ConditionFalse))
				Expect(ready.Reason).To(Equal("Errored"))
			})

			It("should have set the Errored condition as true", func() {
				errored := dc.Status.Conditions[1]

				Expect(errored.Type).To(Equal("Errored"))
				Expect(errored.Status).To(Equal(metav1.ConditionTrue))
				Expect(errored.Reason).To(Equal("test reason"))
				Expect(errored.Message).To(Equal("test message"))
			})
		})

		Context("with failed status update", func() {
			BeforeEach(func() {
				c.EXPECT().Update(context.TODO(), dc).Return(errors.New("some error"))
			})

			It("should return an error", func() {
				err := u.SetConditionsErrored(context.TODO(), dc, "test reason", "test message")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

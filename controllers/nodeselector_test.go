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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hlaiv1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	"github.com/HabanaAI/habana-ai-operator/internal/client"
)

const (
	testNodeName = "test-node"
)

var _ = Describe("NodeSelectorValidator", func() {
	Describe("CheckDeviceConfigForConflictingNodeSelector", func() {
		node := makeTestNode(labelled(map[string]string{"matching": "label"}))
		dc := makeTestDeviceConfig(nodeSelector(node.Labels))
		ctx := context.TODO()

		Context("with a client listing error", func() {
			var (
				gCtrl *gomock.Controller
				c     *client.MockClient
				nsv   *nodeSelectorValidator
			)

			nonconflictingDC := makeTestDeviceConfig(named("nonconflictingDC"))

			s := scheme.Scheme
			Expect(hlaiv1alpha1.AddToScheme(s)).ToNot(HaveOccurred())

			BeforeEach(func() {
				gCtrl = gomock.NewController(GinkgoT())
				c = client.NewMockClient(gCtrl)

				nsv = NewNodeSelectorValidator(c)

				gomock.InOrder(
					c.EXPECT().
						List(ctx, gomock.Any()).
						Return(apierrors.NewServiceUnavailable("Service unavailable")),
				)
			})

			It("should not requeue or return an error", func() {
				err := nsv.CheckDeviceConfigForConflictingNodeSelector(ctx, nonconflictingDC)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Service unavailable"))
			})
		})

		Context("with an invalid/conflicting nodeSelector", func() {
			It("should return an error", func() {
				conflictingDC := makeTestDeviceConfig(named("conflictingDC"), nodeSelector(node.Labels))

				s := scheme.Scheme
				Expect(hlaiv1alpha1.AddToScheme(s)).ToNot(HaveOccurred())

				c := fake.
					NewClientBuilder().
					WithScheme(s).
					WithObjects(node, dc, conflictingDC).
					Build()
				nsv := NewNodeSelectorValidator(c)

				err := nsv.CheckDeviceConfigForConflictingNodeSelector(context.TODO(), conflictingDC)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a valid nodeSelector", func() {
			It("should not return an error", func() {
				nonconflictingDC := makeTestDeviceConfig(named("nonconflictingDC"))

				s := scheme.Scheme
				Expect(hlaiv1alpha1.AddToScheme(s)).ToNot(HaveOccurred())

				c := fake.
					NewClientBuilder().
					WithScheme(s).
					WithObjects(node, dc, nonconflictingDC).
					Build()
				nsv := NewNodeSelectorValidator(c)

				err := nsv.CheckDeviceConfigForConflictingNodeSelector(context.TODO(), nonconflictingDC)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("getDeviceConfigSelectedNodes", func() {
		Context("with a valid nodeSelector", func() {
			It("should returned the selected nodes", func() {
				node := makeTestNode(labelled(map[string]string{"matching": "label"}))
				dc := makeTestDeviceConfig(nodeSelector(node.Labels))

				s := scheme.Scheme
				Expect(hlaiv1alpha1.AddToScheme(s)).ToNot(HaveOccurred())

				c := fake.
					NewClientBuilder().
					WithScheme(s).
					WithObjects(node, dc).
					Build()
				nsv := NewNodeSelectorValidator(c)

				nodeList, err := nsv.getDeviceConfigSelectedNodes(context.TODO(), dc)

				Expect(err).ToNot(HaveOccurred())
				Expect(nodeList.Items).To(HaveLen(1))
				Expect(testNodeName).To(Equal(nodeList.Items[0].Name))
			})
		})
	})
})

var _ = Describe("ContainsDuplicates", func() {
	Context("with a list without duplicates", func() {
		It("should return false", func() {
			arr := []string{"apple", "orange"}

			Expect(containsDuplicates(arr)).To(BeFalse())
		})
	})

	Context("with a list with duplicates", func() {
		It("should return true", func() {
			arr := []string{"apple", "apple"}

			Expect(containsDuplicates(arr)).To(BeTrue())
		})
	})
})

func labelled(labels map[string]string) nodeOptions {
	return func(n *corev1.Node) {
		n.ObjectMeta.Labels = labels
	}
}

type nodeOptions func(*corev1.Node)

func makeTestNode(opts ...nodeOptions) *corev1.Node {
	n := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
	}
	for _, o := range opts {
		o(n)
	}
	return n
}

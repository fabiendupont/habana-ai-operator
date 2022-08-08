package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Main Suite")
}

var _ = Describe("GetEnvWithDefault", func() {
	It("should return an error if the variable is not set", func() {
		ns, err := getWatchNamespace()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("WATCH_NAMESPACE must be set"))
		Expect(ns).To(BeEmpty())
	})

	It("should return the environment variable if it is defined", func() {
		const value = "some-namespace"

		GinkgoT().Setenv("WATCH_NAMESPACE", value)

		ns, err := getWatchNamespace()

		Expect(err).To(BeNil())
		Expect(ns).To(Equal(value))
	})
})

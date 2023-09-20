package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Unit tests for getOCPRange", func() {
	DescribeTable("Providing a valid range of Kubernetes Version",
	func(kubeVersionRange string, expectedOCPResultRange string) {
		OCPResultRange, err := GetOCPRange(kubeVersionRange)
		Expect(err).NotTo(HaveOccurred())
		Expect(OCPResultRange).To(Equal(expectedOCPResultRange))
	},
	PEntry("When providing a single version, using == and no space", "==1.13.0", "4.1"), // TODO: Getting "improper constraint: ==1.13.0"
	PEntry("When providing a single version, using == and space", "== 1.13.0", "4.1"), // TODO: Getting "improper constraint: == 1.13.0"
	Entry("When providing a single version without ==", "1.13.0", "4.1"),
	Entry("When providing a single version without patch", "1.13", "4.1"),
	Entry("When providing a range with a lower limit only, using space", ">= 1.14.0", ">=4.2"),
	Entry("When providing a range with a lower limit only, not using space", ">=1.14.0", ">=4.2"),
	PEntry("When providing a range with an upper limit only, using space", "<= 1.20.0", "<=4.7"), // TODO: current output is 4.1 - 4.7
	PEntry("When providing a range with an upper limit only, not using space", "<=1.20.0", "<=4.7"), // TODO: current output is 4.1 - 4.7
	Entry("When providing a range using '-'", "1.14.0 - 1.20.0", "4.2 - 4.7"),
	Entry("When providing a range using lower and upper limits", ">=1.14.0 <=1.20.0", "4.2 - 4.7"),
	)

	DescribeTable("Providing an invalid range of Kubernetes Version",
	func(kubeVersionRange string) {
		_, err := GetOCPRange(kubeVersionRange)
		Expect(err).To(HaveOccurred())
		// Expect(err).To(MatchError(KuberVersionProcessingError))
	},
	Entry("When providing and unknown version", "0.1.0"),
	PEntry("When providing only a major version", "1"), // TODO: currently getting ">=4.1"
	Entry("When providing an invalid range", "1.20.0 - 1.13.0"),
	)

})
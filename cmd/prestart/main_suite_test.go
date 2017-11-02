package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
	"testing"
)

func TestUpgrader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Prestart Executable Suite")
}

var pathToPreStart string

var _ = BeforeSuite(func() {
	var err error

	pathToPreStart, err = gexec.Build("github.com/cloudfoundry/mariadb_ctrl/cmd/prestart")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

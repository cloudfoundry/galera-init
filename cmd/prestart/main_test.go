package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"os/exec"
	"time"
)

var _ = Describe("MariaDB_Ctrl Prestart", func() {
	Describe("when no Config or Config Path is specified", func() {
		It("panics and emits an error message", func() {
			args := []string{}

			cmd := exec.Command(pathToPreStart, args...)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(2))

			Expect(string(session.Out.Contents())).To(ContainSubstring("No Config or Config Path Specified"))
		})
	})

	Describe("when a valid configPath command line argument is specified", func() {
		It("panics and emits an error message", func() {
			args := []string{
				"-configPath",
				"integration-config.yml",
			}

			cmd := exec.Command(pathToPreStart, args...)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			//Expect(string(session.Out.Contents())).To(ContainSubstring("No Config or Config Path Specified"))
		})
	})
})

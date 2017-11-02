package prestarter_test

import (
	//"os/exec"
	//
	//. "github.com/cloudfoundry/mariadb_ctrl/prestarter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os/exec"
	"github.com/onsi/gomega/gexec"
	"time"
)

var _ = Describe("Prestarter", func() {

	Describe("when no Config or Config Path is specified", func() {
		It("panics and emits an error message", func() {
			args := []string{}

			//cmd := exec.Command(pathToPreStart, args...)
			//session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)


			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(2))

			Expect(string(session.Out.Contents())).To(ContainSubstring("No Config or Config Path Specified"))
		})
	})


	Describe("preStart", func() {
		Context("", func() {
			It("", func() {
				Expect(1).To(Equal(1))
			})
		})
	})
})

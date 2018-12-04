package integration_test

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Process Lifecycle", func() {
	findChildProcess := func() *os.Process {
		grepCmd := exec.Command("pgrep", "mysqld")
		isaacPIDBytes, err := grepCmd.Output()
		Expect(err).NotTo(HaveOccurred())

		isaacPIDStr := string(isaacPIDBytes)
		isaacPID, err := strconv.Atoi(strings.TrimSpace(isaacPIDStr))
		Expect(err).NotTo(HaveOccurred())

		isaac, err := os.FindProcess(isaacPID)
		Expect(err).NotTo(HaveOccurred())
		return isaac
	}

	FContext("When the process starts", func() {
		var (
			exitStatusChan chan int
		)
		BeforeEach(func() {
			exitStatusChan = make(chan int)
		})

		It("galera-init exits when the child mysql process is killed with SIGKILL", func() {
			defer GinkgoRecover()
			abrahamCmd := exec.Command(PathToAbraham, "-configPath", "fixtures/abraham/config.yml")
			abrahamCmd.Stdout = os.Stdout
			abrahamCmd.Stderr = os.Stderr

			err := abrahamCmd.Start()
			Expect(err).NotTo(HaveOccurred())

			// Need to take a quick nap...
			time.Sleep(5 * time.Second)
			isaac := findChildProcess()

			go func() {
				exitStatus := retrieveExitStatus(abrahamCmd.Wait())
				exitStatusChan <- exitStatus
			}()

			// Need to sleep to let the db come up
			time.Sleep(5 * time.Second)
			err = isaac.Signal(syscall.SIGKILL)
			Expect(err).NotTo(HaveOccurred())

			Eventually(exitStatusChan).Should(
				Receive(Equal(int(syscall.SIGKILL))), "Expected galera-init process to exit with 9, indicating a SIGKILL was received")
		})

		Context("galera-init exits when the child mysql process is killed with SIGTERM", func() {
			It("gracefully shutsdown", func() {
				abrahamCmd := exec.Command(PathToAbraham, "-configPath", "fixtures/abraham/config.yml")
				abrahamCmd.Stdout = os.Stdout
				abrahamCmd.Stderr = os.Stderr

				err := abrahamCmd.Start()
				Expect(err).NotTo(HaveOccurred())

				// Need to take a quick nap...
				time.Sleep(5 * time.Second)
				isaac := findChildProcess()

				go func() {
					defer GinkgoRecover()
					exitStatus := abrahamCmd.Wait()
					Expect(exitStatus).To(BeNil())
					if exitStatus == nil {
						exitStatusChan <- 0
					}
				}()

				// Need to sleep to let the db come up
				time.Sleep(5 * time.Second)
				err = isaac.Signal(syscall.SIGTERM)
				Expect(err).NotTo(HaveOccurred())

				Eventually(exitStatusChan).Should(Receive(Equal(0)), "Expected galera-init process to exit with 15, indicating a SIGTERM was received")
			})
		})

		Context("mysqld exits when the parent galera-init process is killed with SIGTERM ", func() {
			FIt("gracefully shutsdown", func() {
				abrahamCmd := exec.Command(PathToAbraham, "-configPath", "fixtures/abraham/config.yml")
				abrahamCmd.Stdout = os.Stdout
				abrahamCmd.Stderr = os.Stderr

				err := abrahamCmd.Start()
				Expect(err).NotTo(HaveOccurred())

				// Need to sleep to let the db come up
				// Should we inspect the mysqld.log instead
				time.Sleep(5 * time.Second)
				err = abrahamCmd.Process.Signal(syscall.SIGTERM)
				Expect(err).NotTo(HaveOccurred())

				var exitError error
				Eventually(func() error {
					grepCmd := exec.Command("pgrep", "mysqld")
					_, exitError = grepCmd.Output()
					return exitError
				}, 20*time.Second, 1*time.Second).Should(HaveOccurred())

				Expect(retrieveExitStatus(exitError)).To(Equal(1))
			})
		})

		Context("galera-init fails to bootstrap process", func() {
			stateFile := "testStateFileLocation"

			BeforeEach(func() {
				file, err := os.Create(stateFile)
				Expect(err).NotTo(HaveOccurred())
				file.WriteString("CLUSTERED")
				Expect(err).NotTo(HaveOccurred())
				err = os.Chmod(stateFile, 0400)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.Remove(stateFile)
			})

			It("shutdowns mysqld", func() {
				defer GinkgoRecover()

				abrahamCmd := exec.Command(PathToAbraham, "-configPath", "fixtures/abraham/config.yml")
				abrahamCmd.Stdout = os.Stdout
				abrahamCmd.Stderr = os.Stderr

				err := abrahamCmd.Start()
				Expect(err).NotTo(HaveOccurred())

				go func() {
					exitStatus := retrieveExitStatus(abrahamCmd.Wait())
					exitStatusChan <- exitStatus
				}()

				Eventually(exitStatusChan).Should(Receive(Equal(int(1))), "Expected galera-init process to exit with 1, indicating an error inside galera-init, not mysqld")

				Eventually(func() bool {
					grepCmd := exec.Command("pgrep", "mysqld")
					isaacPIDBytes, err := grepCmd.Output()
					Expect(err).To(HaveOccurred())
					Expect(retrieveExitStatus(err)).To(Equal(1))
					return len(isaacPIDBytes) == 0
				}).Should(BeTrue())
			})
		})
	})
})

func retrieveExitStatus(err error) int {
	return err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()
}

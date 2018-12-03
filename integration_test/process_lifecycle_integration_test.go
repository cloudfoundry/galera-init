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
				exitStatus := abrahamCmd.Wait().(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()
				exitStatusChan <- exitStatus
			}()

			// Need to sleep to let the db come up
			time.Sleep(5 * time.Second)
			err = isaac.Signal(syscall.SIGKILL)
			Expect(err).NotTo(HaveOccurred())

			var exitStatus int

			Eventually(func() int {
				exitStatus = <-exitStatusChan
				return exitStatus
			}).ShouldNot(Equal(0))

			Expect(exitStatus).Should(Equal(int(syscall.SIGKILL)), "Expected galera-init process to exit with 9, indicating a SIGKILL was received")
		})

		Context("galera-init exits when the child mysql process is killed with SIGTERM ", func() {
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
					exitStatus := abrahamCmd.Wait() // .(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()
					Expect(exitStatus).To(BeNil())
					if exitStatus == nil {
						exitStatusChan <- 0
					}
				}()

				// Need to sleep to let the db come up
				time.Sleep(5 * time.Second)
				err = isaac.Signal(syscall.SIGTERM)
				Expect(err).NotTo(HaveOccurred())

				var exitStatus int

				Eventually(func() int {
					exitStatus = <-exitStatusChan
					return exitStatus
				}).Should(Equal(0))

				Expect(exitStatus).Should(Equal(int(0)), "Expected galera-init process to exit with 15, indicating a SIGTERM was received")
			})
		})

		Context("galera-init fails to bootstrap process", func() {
			stateFile := "testStateFileLocation"
			BeforeEach(func() {
				os.Remove(stateFile)
			})

			FIt("shutdowns mysqld", func() {
				defer GinkgoRecover()

				file, err := os.Create(stateFile)
				Expect(err).NotTo(HaveOccurred())
				file.WriteString("CLUSTERED")
				Expect(err).NotTo(HaveOccurred())
				err = os.Chmod(stateFile, 0400)
				Expect(err).NotTo(HaveOccurred())

				abrahamCmd := exec.Command(PathToAbraham, "-configPath", "fixtures/abraham/config.yml")
				abrahamCmd.Stdout = os.Stdout
				abrahamCmd.Stderr = os.Stderr

				err = abrahamCmd.Start()
				Expect(err).NotTo(HaveOccurred())

				go func() {
					exitStatus := abrahamCmd.Wait().(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()
					exitStatusChan <- exitStatus
				}()

				exitStatus := <-exitStatusChan
				Expect(exitStatus).Should(Equal(int(1)), "Expected galera-init process to exit with 1, indicating an error inside galera-init, not mysqld")

				Eventually(func() bool {
					grepCmd := exec.Command("pgrep", "mysqld")
					isaacPIDBytes, err := grepCmd.Output()
					Expect(err).To(HaveOccurred())
					Expect(err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()).To(Equal(1))
					return len(isaacPIDBytes) == 0
				}).Should(BeTrue())
			})
		})
	})
})

package integration_test

import (
	"fmt"
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
		fmt.Println("isaac pid string: ", isaacPIDStr)
		isaacPID, err := strconv.Atoi(strings.TrimSpace(isaacPIDStr))
		Expect(err).NotTo(HaveOccurred())

		isaac, err := os.FindProcess(isaacPID)
		Expect(err).NotTo(HaveOccurred())
		return isaac
	}

	Context("When the process starts", func() {
		var (
			sigtermExitCode int
		)

		BeforeEach(func() {
			sigtermExitCode = 128 + 15 // 128 to reach the process terminated range of exit codes, 15 for sigterm
		})

		It("galera-init exits when the child mysql process is killed", func() {

			abrahamCmd := exec.Command(PathToAbraham, "-configPath", "fixtures/abraham/config.yml")
			abrahamCmd.Stdout = os.Stdout

			err := abrahamCmd.Start()
			Expect(err).NotTo(HaveOccurred())

			// Need to take a quick nap...
			time.Sleep(10 * time.Second)
			isaac := findChildProcess()

			var exitStatus int
			go func() {
				// Ooooooh yeahhh, check out how idiotic -- I mean, how idiomatic this is!
				exitStatus = abrahamCmd.Wait().(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()
			}()

			err = isaac.Signal(syscall.SIGTERM)
			Expect(err).NotTo(HaveOccurred())

			// Processes that exit due to a signal have exit codes above 128 (128 + SIGNAL)
			// Let's check for 128 + 15 (sigterm)
			Eventually(func() int {
				return exitStatus
			}, 5*time.Second).Should(Equal(sigtermExitCode), "Expected galera-init process to exit with 143, indicating a SIGTERM was received")

			fmt.Printf("ExitStatus: %d\n", exitStatus)

		})
	})
})

package docker_test

import (
	"database/sql"
	"encoding/json"
	"github.com/cloudfoundry/galera-init/config"
	"github.com/fsouza/go-dockerclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pkg/errors"
	"log"
	"syscall"
	"time"

	. "github.com/cloudfoundry/galera-init/docker/test_helpers"
)

func createGaleraContainer(name string, cfg config.Config) (*docker.Container, error) {
	marshalledConfig, err := json.Marshal(&cfg)
	if err != nil {
		return nil, errors.New("failed to marshal configuration")
	}

	return RunContainer(
		dockerClient,
		name+"."+sessionID,
		AddExposedPorts(pxcMySQLPort),
		AddBinds(
			initSh+":/usr/local/bin/init.sh",
			galeraInitPath+":/usr/local/bin/galera-init",
			galeraInitConfigPath+":/usr/local/etc/galera-init-config.yml",
			mysqlConfigPath+":/etc/mysql/my.cnf",
		),
		AddEnvVars(
			"CONFIG="+string(marshalledConfig),
		),
		WithEntrypoint("init.sh"),
		WithImage(pxcDockerImage),
		WithNetwork(dockerNetwork),
	)
}

var _ = Describe("galera-init integration", func() {
	var (
		cfg config.Config
	)

	BeforeEach(func() {
		cfg = config.Config{
			LogFileLocation: "testPath",
			Db: config.DBHelper{
				UpgradePath:        "mysql_upgrade",
				User:               "root",
				PreseededDatabases: nil,
				Socket:             "/var/run/mysqld/mysqld.sock",
			},
			Manager: config.StartManager{
				StateFileLocation:    "/var/lib/mysql/node_state.txt",
				GrastateFileLocation: "/var/lib/mysql/grastate.dat",
				ClusterIps: []string{
					"mysql0." + sessionID,
				},
				BootstrapNode:       true,
				ClusterProbeTimeout: 10,
			},
			Upgrader: config.Upgrader{
				PackageVersionFile:      "testPackageVersionFile",
				LastUpgradedVersionFile: "testLastUpgradedVersionFile",
			},
		}
	})

	When("Starting a single node", func() {
		var (
			singleNodeContainer *docker.Container
			db                  *sql.DB
		)

		BeforeEach(func() {
			var err error

			singleNodeContainer, err = createGaleraContainer("mysql0", cfg)
			Expect(err).NotTo(HaveOccurred())

			db, err = ContainerDBConnection(singleNodeContainer, pxcMySQLPort)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if db != nil {
				_ = db.Close()
			}

			if singleNodeContainer != nil {
				//Expect(RemoveContainer(dockerClient, singleNodeContainer)).To(Succeed())
			}
		})

		It("can manage a MySQL process", func() {
			By("eventually bringing MySQL online", func() {

				Eventually(func() error {
					log.Println("Attempting to ping MySQL in docker container...")
					return db.Ping()
				}, "30s", "1s").Should(Succeed())
			})

			By("Exiting gracefully when terminated", func() {
				err := dockerClient.KillContainer(docker.KillContainerOptions{
					ID:     singleNodeContainer.ID,
					Signal: docker.SIGTERM,
				})
				Expect(err).NotTo(HaveOccurred())

				output := gbytes.NewBuffer()

				go dockerClient.Logs(docker.LogsOptions{
					Container:    singleNodeContainer.ID,
					OutputStream: output,
					ErrorStream:  output,
					Follow:       true,
					Stdout:       true,
					Stderr:       true,
					Tail:         "0",
				})

				Eventually(output, "30s", "1s").Should(gbytes.Say(`\[Note\] mysqld: Shutdown complete`))

				Eventually(func() bool {
					log.Println("Checking container state")
					container, _ := dockerClient.InspectContainer(singleNodeContainer.ID)
					log.Printf("state: %s", container.State.String())
					return !container.State.Running
				}, "30s", "1s").Should(BeTrue())
			})
		})

		It("will terminate with an error when mysqld terminate", func() {
			By("eventually bringing MySQL online", func() {
				Eventually(func() error {
					log.Println("Attempting to ping MySQL in docker container...")
					return db.Ping()
				}, "30s", "1s").Should(Succeed())
			})

			// temporary hack: we don't really know when galera-init has finished
			// maybe we should tail the container locks and look for the "Bootstrapping done"
			// message
			time.Sleep(5*time.Second)

			By("terminating the mysql server process without prejudice", func() {
				_, err := TerminateMySQLD(dockerClient, singleNodeContainer)
				Expect(err).NotTo(HaveOccurred())

			})

			Eventually(func() (exitCode int, err error) {
				container, err := dockerClient.InspectContainer(singleNodeContainer.ID)
				if err != nil {
					return 0, err
				}
				return container.State.ExitCode, err
			}, "30s", "1s").Should(Equal(int(syscall.SIGKILL)))
		})
	})
})

package docker_test

import (
	"github.com/go-sql-driver/mysql"
	"github.com/nu7hatch/gouuid"
	"github.com/onsi/gomega/gexec"
	"log"
	"os"
	"path/filepath"
	"testing"

	. "github.com/cloudfoundry/galera-init/docker/test_helpers"

	"github.com/fsouza/go-dockerclient"
	_ "github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDocker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Docker Suite")
}

const (
	pxcDockerImage = "percona/percona-xtradb-cluster:5.7"
	pxcMySQLPort   = "3306/tcp"
)

var (
	dockerClient         *docker.Client
	dockerNetwork        *docker.Network
	sessionID            string
	galeraInitPath       string
	galeraInitConfigPath string
	mysqlConfigPath      string
	initSh               string
)

var _ = BeforeSuite(func() {
	log.SetOutput(GinkgoWriter)

	var err error
	dockerClient, err = docker.NewClientFromEnv()
	Expect(err).NotTo(HaveOccurred())

	Expect(PullImage(dockerClient, pxcDockerImage)).To(Succeed())

	dockerNetwork, err = CreateNetwork(dockerClient, "mysql-net."+sessionID)
	Expect(err).NotTo(HaveOccurred())

	galeraInitPath, err = gexec.BuildWithEnvironment(
		"github.com/cloudfoundry/galera-init/cmd/start/",
		[]string{
			"GOOS=linux",
			"GOARCH=amd64",
		},
	)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.Rename(galeraInitPath, "build/galera-init")).To(Succeed())
	galeraInitPath, err = filepath.Abs(filepath.Join("build", "galera-init"))
	Expect(err).NotTo(HaveOccurred())

	mysql.SetLogger(log.New(GinkgoWriter, "[mysql] ", log.Ldate|log.Ltime|log.Lshortfile))
})


var _ = AfterSuite(func() {
	Expect(dockerClient.RemoveNetwork(dockerNetwork.ID)).To(Succeed())
})

var _ = BeforeEach(func() {
	uuid, err := uuid.NewV4()
	Expect(err).NotTo(HaveOccurred())
	sessionID = uuid.String()

	myCnf := filepath.Join("fixtures", "my.cnf")
	mysqlConfigPath, err = filepath.Abs(myCnf)
	Expect(err).NotTo(HaveOccurred())

	galeraInitConfigPath, err = filepath.Abs(filepath.Join("fixtures", "galera-init-config.yml"))
	Expect(err).NotTo(HaveOccurred())

	initSh, err = filepath.Abs(filepath.Join("fixtures", "init.sh"))

})

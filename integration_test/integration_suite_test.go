package integration_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/service-config"
)

var testConfig TestDBConfig
var serviceConfig *service_config.ServiceConfig

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Integration Test Suite")
}

type TestDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

var (
	PathToIsaac   string
	PathToAbraham string
	newPath       string
	oldPath       string
)

var _ = BeforeSuite(func() {

	serviceConfig = service_config.New()

	//Use default options rather than throw error if env variables are blank
	if os.Getenv("CONFIG") == "" && os.Getenv("CONFIG_PATH") == "" {
		os.Setenv("CONFIG", "{}")
	}
	serviceConfig.AddDefaults(TestDBConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "",
	})

	err := serviceConfig.Read(&testConfig)
	Expect(err).NotTo(HaveOccurred())

	PathToIsaac, err = gexec.Build("github.com/cloudfoundry/galera-init/integration_test/fixtures/isaac/")
	Expect(err).NotTo(HaveOccurred())
	fmt.Println(PathToIsaac)

	PathToAbraham, err = gexec.Build("github.com/cloudfoundry/galera-init/cmd/start/")
	Expect(err).NotTo(HaveOccurred())

	dirname := path.Dir(PathToIsaac)
	err = os.Rename(PathToIsaac, dirname+"/mysqld")
	Expect(err).NotTo(HaveOccurred())

	oldPath = os.Getenv("PATH")
	newPath = fmt.Sprintf("%s:%s", dirname, oldPath)
	os.Setenv("PATH", newPath)
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	os.Setenv("PATH", oldPath)
})

package integration_test

import (
	"database/sql"
	"fmt"
	// docker "github.com/cloudfoundry/galera-init/vendor/github.com/fsouza/go-dockerclient"
	"strings"

	"code.cloudfoundry.org/lager/lagertest"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/go-sql-driver/mysql"
	"github.com/nu7hatch/gouuid"

	"github.com/cloudfoundry/galera-init/config"
	"github.com/cloudfoundry/galera-init/db_helper"
	"github.com/cloudfoundry/galera-init/integration_test/test_helpers"
	"github.com/cloudfoundry/galera-init/os_helper/os_helperfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var testConfig TestDBConfig

type TestDBConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
}

var _ = Describe("DB Helper", func() {
	var (
		galeraNode *docker.Container
		db         *sql.DB
	)

	BeforeEach(func() {
		var err error

		for i := 0; i < 3; i++ {

			galeraNode, err = test_helpers.RunContainer(
				dockerClient,
				"mysql0."+sessionID,
				test_helpers.WithImage(pxcDockerImage),
				test_helpers.AddEnvVars(
					"MYSQL_ALLOW_EMPTY_PASSWORD=1",
					"CLUSTER_NAME=db-helper-cluster",
				),
				test_helpers.WithCmd("--pxc-strict-mode=MASTER"),
			)
			Expect(err).NotTo(HaveOccurred())

			rootDsn := fmt.Sprintf("root@tcp(127.0.0.1:%s)/", test_helpers.HostPort(pxcMySQLPort, galeraNode))
			db, err = sql.Open("mysql", rootDsn)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() string {
				result := PingAndDeadlockCheck(db, galeraNode)
				fmt.Printf("result spam: %v\n", result)
				return result
			}, "1m", "2s").Should(Not(ContainSubstring("truckin")))
			errString := PingAndDeadlockCheck(db, galeraNode)
			if errString == "our flake" {
				fmt.Printf("\n\n\n*********************************\n*********************************\nSHAMALAMADINGDONG]\n\n*************************************\n\n\n\n\n")
				//blow away container
				if galeraNode != nil {
					Expect(test_helpers.RemoveContainer(dockerClient, galeraNode)).To(Succeed())
				}
				continue
			}
			break
		}

		testConfig = TestDBConfig{
			Host:     "127.0.0.1:" + test_helpers.HostPort(pxcMySQLPort, galeraNode),
			User:     "root",
			Password: "",
			DBName:   "",
		}

		rootDsn := fmt.Sprintf("root@tcp(127.0.0.1:%s)/", test_helpers.HostPort(pxcMySQLPort, galeraNode))

		//override db connection to use test DB
		db_helper.OpenDBConnection = func(config *config.DBHelper) (*sql.DB, error) {
			return sql.Open("mysql", rootDsn)
		}
	})

	AfterEach(func() {
		if galeraNode != nil {
			Expect(test_helpers.RemoveContainer(dockerClient, galeraNode)).To(Succeed())
		}
	})

	Describe("Seed", func() {
		var (
			helper     *db_helper.GaleraDBHelper
			fakeOs     *os_helperfakes.FakeOsHelper
			testLogger lagertest.TestLogger
			logFile    string
			dbConfig   *config.DBHelper
		)

		BeforeEach(func() {
			// MySQL mandates usernames are <= 32 chars
			user0 := getUUIDWithPrefix("GALERA_INIT")[:32]
			user1 := getUUIDWithPrefix("GALERA_INIT")[:32]
			databaseA := getUUIDWithPrefix("GALERA_INIT_DB")
			databaseB := getUUIDWithPrefix("GALERA_INIT_DB")

			dbConfig = &config.DBHelper{
				User:     testConfig.User,
				Password: testConfig.Password,
				// Same user for multiple databases, and same database for multiple users
				PreseededDatabases: []config.PreseededDatabase{
					{
						DBName:   databaseA,
						User:     user0,
						Password: "password0",
					},
					{
						DBName:   databaseB,
						User:     user0,
						Password: "password0",
					},
					{
						DBName:   databaseB,
						User:     user1,
						Password: "password1",
					},
				},
			}

			fakeOs = new(os_helperfakes.FakeOsHelper)
			testLogger = *lagertest.NewTestLogger("db_helper")
			logFile = "/log-file.log"
		})

		JustBeforeEach(func() {
			helper = db_helper.NewDBHelper(
				fakeOs,
				dbConfig,
				logFile,
				testLogger,
			)
		})

		AfterEach(func() {
			defer db.Close()

			for _, preseededDB := range dbConfig.PreseededDatabases {
				_, err := db.Exec(
					fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", preseededDB.DBName))
				Expect(err).NotTo(HaveOccurred())

				_, err = db.Exec(
					fmt.Sprintf("DROP USER IF EXISTS '%s'", preseededDB.User))
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("Seeding databases and users", func() {
			const mysqlAccessDenied uint16 = 1044
			var ensureSeedSucceeds = func() {
				err := helper.Seed()
				Expect(err).NotTo(HaveOccurred())

				for _, preseededDB := range dbConfig.PreseededDatabases {
					//check that DB exists
					dbRows, err := db.Query(fmt.Sprintf("SHOW DATABASES LIKE '%s'", preseededDB.DBName))
					Expect(err).NotTo(HaveOccurred())
					Expect(dbRows.Err()).NotTo(HaveOccurred())
					Expect(dbRows.Next()).To(BeTrue(), fmt.Sprintf("Expected DB to exist: %s", preseededDB.DBName))

					//check that user can login to DB
					userDb, err := sql.Open("mysql", fmt.Sprintf(
						"%s:%s@tcp(%s)/%s",
						preseededDB.User,
						preseededDB.Password,
						testConfig.Host,
						preseededDB.DBName,
					))

					Expect(err).NotTo(HaveOccurred())
					defer userDb.Close()

					//check that user has CREATE permission
					_, err = userDb.Exec("CREATE TABLE testTable ( ID int )")
					Expect(err).NotTo(HaveOccurred())

					//check that user has INSERT permission
					_, err = userDb.Exec("INSERT INTO testTable (ID) VALUES (1)")

					//check that user does not have LOCK TABLES permission
					_, err = userDb.Exec("LOCK TABLES testTable READ")
					Expect(err).To(HaveOccurred())
					e, found := err.(*mysql.MySQLError)
					Expect(found).To(BeTrue())
					Expect(e.Number).To(Equal(mysqlAccessDenied))

					//check that user has DROP permission
					_, err = userDb.Exec("DROP TABLE testTable")
					Expect(err).NotTo(HaveOccurred())
				}
			}

			It("seeds databases and users the first time", func() {
				ensureSeedSucceeds()
			})

			It("updates users if they are re-seeded with different passwords during a subsequent deploy", func() {
				ensureSeedSucceeds()
				dbConfig.PreseededDatabases[0].Password = "reseeded-password0"
				dbConfig.PreseededDatabases[1].Password = "reseeded-password0"
				dbConfig.PreseededDatabases[2].Password = "reseeded-password1"
				ensureSeedSucceeds()
			})

			Context("when database name contains a hyphen", func() {
				BeforeEach(func() {
					dbNameWithHyphen := getUUIDWithPrefix("GALERA_INIT_DB")
					dbNameWithHyphen = strings.Replace(dbNameWithHyphen, "_", "-", -1)

					dbConfig.PreseededDatabases[0].DBName = dbNameWithHyphen
				})

				It("seeds databases and users", func() {
					ensureSeedSucceeds()
				})
			})

			Context("when user name contains a hyphen", func() {
				BeforeEach(func() {
					userWithHyphen := getUUIDWithPrefix("GALERA_INIT")[:16]
					userWithHyphen = strings.Replace(userWithHyphen, "_", "-", -1)

					dbConfig.PreseededDatabases[0].User = userWithHyphen
				})

				It("seeds databases and users", func() {
					ensureSeedSucceeds()
				})
			})
		})
	})
})

func getUUIDWithPrefix(prefix string) string {
	id, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	idString := fmt.Sprintf("%s_%s", prefix, id.String())
	// mysql does not like hyphens in DB names
	return strings.Replace(idString, "-", "_", -1)
}

func PingAndDeadlockCheck(db *sql.DB, galeraNode *docker.Container) string {
	err := db.Ping()
	fmt.Printf("the err: %v\n", err)
	if err == nil {
		return "gucci"
	}

	mysqlErrLogContents, err := test_helpers.FetchContainerFileContents(
		dockerClient,
		galeraNode,
		"/var/log/mysql/mysql.err.log",
	)

	fmt.Printf("The stuff \n: %v", mysqlErrLogContents)
	if strings.Contains(err.Error(), "Could not find the file /var/log/mysql/mysql.err.log") {
		return "keep on truckin'"
	}
	Expect(err).NotTo(HaveOccurred())

	if strings.Contains(mysqlErrLogContents,"Deadlock found when trying to get lock; try restarting transaction"){
		//blow away the container, loop again
		return "our flake"
	}

	//other error OR not ready

	return "keep on truckin'"

}

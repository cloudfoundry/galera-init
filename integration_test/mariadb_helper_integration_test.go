package integration_test

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/cloudfoundry/mariadb_ctrl/mariadb_helper"
	os_fakes "github.com/cloudfoundry/mariadb_ctrl/os_helper/fakes"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MariaDB Helper", func() {
	var (
		helper     *mariadb_helper.MariaDBHelper
		fakeOs     *os_fakes.FakeOsHelper
		testLogger lagertest.TestLogger
		logFile    string
		config     mariadb_helper.Config
		db         *sql.DB
	)

	BeforeEach(func() {
		fakeOs = new(os_fakes.FakeOsHelper)
		testLogger = *lagertest.NewTestLogger("mariadb_helper")
		logFile = "/log-file.log"

		// MySQL mandates usernames are <= 16 chars
		user0 := getUUIDWithPrefix("MARIADB")[:16]
		user1 := getUUIDWithPrefix("MARIADB")[:16]

		config = mariadb_helper.Config{
			User:     "root",
			Password: "password",
			PreseededDatabases: []mariadb_helper.PreseededDatabase{
				mariadb_helper.PreseededDatabase{
					DBName:   getUUIDWithPrefix("MARIADB_CTRL_DB"),
					User:     user0,
					Password: "password0",
				},
				mariadb_helper.PreseededDatabase{
					DBName:   getUUIDWithPrefix("MARIADB_CTRL_DB"),
					User:     user0,
					Password: "password0",
				},
				mariadb_helper.PreseededDatabase{
					DBName:   getUUIDWithPrefix("MARIADB_CTRL_DB"),
					User:     user1,
					Password: "password1",
				},
			},
		}

		helper = mariadb_helper.NewMariaDBHelper(
			fakeOs,
			config,
			logFile,
			testLogger,
		)

		var err error
		db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@/", config.User, config.Password))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {

		defer db.Close()

		for _, preseededDB := range config.PreseededDatabases {
			_, err := db.Exec(
				fmt.Sprintf("DROP DATABASE IF EXISTS %s", preseededDB.DBName))
			testLogger.Error("Error cleaning up test DB's", err)

			_, err = db.Exec(
				fmt.Sprintf("DROP USER %s", preseededDB.User))
			testLogger.Error("Error cleaning up test users", err)
		}
	})

	It("seeds databases and users", func() {
		err := helper.Seed()
		Expect(err).NotTo(HaveOccurred())

		for _, preseededDB := range config.PreseededDatabases {
			//check that DB exists
			dbRows, err := db.Query(fmt.Sprintf("SHOW DATABASES LIKE '%s'", preseededDB.DBName))
			Expect(err).NotTo(HaveOccurred())
			Expect(dbRows.Err()).NotTo(HaveOccurred())
			Expect(dbRows.Next()).To(BeTrue(), fmt.Sprintf("Expected DB to exist: %s", preseededDB.DBName))

			//check that user can login to DB
			userDb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/%s",
				preseededDB.User,
				preseededDB.Password,
				preseededDB.DBName))
			Expect(err).NotTo(HaveOccurred())
			defer userDb.Close()

			//check that user has permission to create a table
			_, err = userDb.Exec("CREATE TABLE testTable ( ID int )")
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

func getUUIDWithPrefix(prefix string) string {
	id, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	idString := fmt.Sprintf("%s_%s", prefix, id.String())
	// mysql does not like hyphens in DB names
	return strings.Replace(idString, "-", "_", -1)
}

package upgrader

import (
	"regexp"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/pkg/errors"

	"github.com/cloudfoundry/galera-init/config"
	"github.com/cloudfoundry/galera-init/db_helper"
	"github.com/cloudfoundry/galera-init/os_helper"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Upgrader
type Upgrader interface {
	Upgrade() error
	NeedsUpgrade() (bool, error)
}

type upgrader struct {
	osHelper os_helper.OsHelper
	config   config.Upgrader
	logger   lager.Logger
	dbHelper db_helper.DBHelper
}

var (
	DBReachablePollingAttempts = 30
	DBReachablePollingDelay    = 10 * time.Second
)

func NewUpgrader(
	osHelper os_helper.OsHelper,
	config config.Upgrader,
	logger lager.Logger,
	dbHelper db_helper.DBHelper) Upgrader {

	return upgrader{
		osHelper: osHelper,
		config:   config,
		logger:   logger,
		dbHelper: dbHelper,
	}
}

func (u upgrader) Upgrade() error {
	u.logger.Info("starting-mysqld-for-upgrade")
	cmd, err := u.dbHelper.StartMysqldForUpgrade()
	if err != nil {
		return err
	}

	mysqldExitChan := u.osHelper.WaitForCommand(cmd)

	if err := u.waitUntilMySQLReachable(); err != nil {
		return err
	}

	u.logger.Info("mysql-upgrade-starting")
	output, upgrade_err := u.dbHelper.Upgrade()

	if upgrade_err != nil {
		acceptableErrorsCompiled, _ := regexp.Compile(
			"already upgraded|Unknown command|WSREP has not yet prepared node")

		if acceptableErrorsCompiled.MatchString(output) {
			u.logger.Info(
				"output string matches acceptable errors - continuing startup.",
				lager.Data{"upgradeErr": upgrade_err, "upgradeOutput": output},
			)
		} else {
			u.logger.Info(
				"output string does not match acceptable errors - aborting startup.",
				lager.Data{"upgradeErr": upgrade_err, "upgradeOutput": output},
			)
			err = upgrade_err
		}
	} else {
		u.logger.Info("mysql-upgrade-complete", lager.Data{
			"upgradeOutput": output,
		})
	}

	u.logger.Info("stopping-upgrade-mysqld")
	u.stopStandaloneDatabaseSynchronously()

	if mysqldErr := <-mysqldExitChan; mysqldErr != nil {
		return errors.Wrap(mysqldErr, `mysqld failed during upgrade`)
	}

	u.logger.Info("mysqld-stopped")

	if err != nil {
		return err
	}

	return nil
}

func (u upgrader) waitUntilMySQLReachable() error {
	u.logger.Info("wait-for-upgrade-mysqld", lager.Data{
		"state": "starting",
	})
	for tries := 0; tries < DBReachablePollingAttempts; tries++ {
		if u.dbHelper.IsDatabaseReachable() {
			u.logger.Info("wait-for-upgrade-mysqld", lager.Data{
				"state": "ready",
			})

			return nil
		}

		u.logger.Info("wait-for-upgrade-mysqld", lager.Data{
			"state": "polling",
		})
		u.osHelper.Sleep(DBReachablePollingDelay)
	}

	u.logger.Info("wait-for-upgrade-mysqld", lager.Data{
		"state": "timeout",
	})
	return errors.New("Database is not reachable after 30 tries.")
}

func (u upgrader) stopStandaloneDatabaseSynchronously() {
	u.dbHelper.StopMysqld()
}

func (u upgrader) NeedsUpgrade() (bool, error) {
	if !u.osHelper.FileExists(u.config.LastUpgradedVersionFile) {
		u.logger.Info(
			"Upgrade required",
			lager.Data{
				"reason":                  "Last Upgraded version file does not exist in data dir",
				"lastUpgradedVersionFile": u.config.LastUpgradedVersionFile,
			})
		return true, nil
	}

	if !u.osHelper.FileExists(u.config.PackageVersionFile) {
		u.logger.Info(
			"Cannot determine whether upgrade is required.",
			lager.Data{
				"reason":             "Package version file does not exist",
				"packageVersionFile": u.config.PackageVersionFile,
			})
		return false, errors.New("DB package is invalid because it is missing the version file.")
	}

	existingVersion, err := u.osHelper.ReadFile(u.config.LastUpgradedVersionFile)
	if err != nil {
		u.logger.Info(
			"Cannot determine whether upgrade is required.",
			lager.Data{
				"reason":                  "Error reading last upgraded version file",
				"lastUpgradedVersionFile": u.config.LastUpgradedVersionFile,
				"err":                     err,
			})
		return false, errors.New("Could not read last upgraded version file in the data dir.")
	}

	packageVersion, err := u.osHelper.ReadFile(u.config.PackageVersionFile)
	if err != nil {
		u.logger.Info(
			"Cannot determine whether upgrade is required.",
			lager.Data{
				"reason":             "Error reading package version file",
				"packageVersionFile": u.config.PackageVersionFile,
				"err":                err,
			})
		return false, errors.New("DB package is invalid because the version file is not readable.")
	}

	if strings.TrimSpace(existingVersion) != strings.TrimSpace(packageVersion) {
		u.logger.Info("Need to upgrade to latest version.")
		return true, nil
	}
	u.logger.Info("Already upgraded to latest version, starting normally.")
	return false, nil
}

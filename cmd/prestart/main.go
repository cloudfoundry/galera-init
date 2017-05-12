package main

import (
	"io/ioutil"
	"os"
	"strconv"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/mariadb_ctrl/cluster_health_checker"
	"github.com/cloudfoundry/mariadb_ctrl/config"
	"github.com/cloudfoundry/mariadb_ctrl/mariadb_helper"
	"github.com/cloudfoundry/mariadb_ctrl/os_helper"
	"github.com/cloudfoundry/mariadb_ctrl/start_manager"
	"github.com/cloudfoundry/mariadb_ctrl/start_manager/node_runner"
	"github.com/cloudfoundry/mariadb_ctrl/start_manager/node_starter"
	"github.com/cloudfoundry/mariadb_ctrl/upgrader"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	cfg, err := config.NewConfig(os.Args)
	if err != nil {
		cfg.Logger.Fatal("Error creating config", err)
		return
	}

	err = cfg.Validate()
	if err != nil {
		cfg.Logger.Fatal("Error validating config", err)
		return
	}

	sigRunner := newRunner(cfg)

	process := ifrit.Background(sigRunner)

	select {
	case err = <-process.Wait():
		cfg.Logger.Error("Error starting mysqld", err)
		os.Exit(1)
	case <-process.Ready():
		//continue
	}

	err = writePidFile(cfg)
	if err != nil {
		process.Signal(os.Kill)
		<-process.Wait()

		cfg.Logger.Fatal("Error writing pidfile", err, lager.Data{
			"PidFile": cfg.PidFile,
		})
		return
	}

	cfg.Logger.Info("mariadb_ctrl started, immediately shutting down")

	process.Signal(os.Kill)
	<-process.Wait()

	err = deletePidFile(cfg)
	if err != nil {
		cfg.Logger.Error("Error deleting pidfile", err, lager.Data{
			"pidfile": cfg.PidFile,
		})
	}

	cfg.Logger.Info("Process exited without error.")
}

func writePidFile(cfg *config.Config) error {
	cfg.Logger.Info("Writing pid", lager.Data{
		"pidfile": cfg.PidFile,
	})
	return ioutil.WriteFile(cfg.PidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func deletePidFile(cfg *config.Config) error {
	cfg.Logger.Info("Deleting pidfile", lager.Data{
		"pidfile": cfg.PidFile,
	})
	return os.Remove(cfg.PidFile)
}

func newRunner(cfg *config.Config) ifrit.Runner {
	OsHelper := os_helper.NewImpl()

	DBHelper := mariadb_helper.NewMariaDBHelper(
		OsHelper,
		cfg.Db,
		cfg.LogFileLocation,
		cfg.Logger,
	)

	Upgrader := upgrader.NewUpgrader(
		OsHelper,
		cfg.Upgrader,
		cfg.Logger,
		DBHelper,
	)

	ClusterHealthChecker := cluster_health_checker.NewClusterHealthChecker(
		cfg.Manager.ClusterIps,
		cfg.Manager.ClusterProbeTimeout,
		cfg.Logger,
	)

	NodeStarter := node_starter.NewPreStarter(
		DBHelper,
		OsHelper,
		cfg.Manager,
		cfg.Logger,
		ClusterHealthChecker,
	)

	NodeStartManager := start_manager.New(
		OsHelper,
		cfg.Manager,
		DBHelper,
		Upgrader,
		NodeStarter,
		cfg.Logger,
		ClusterHealthChecker,
	)

	runner := node_runner.NewPrestartRunner(NodeStartManager, cfg.Logger)

	sigRunner := sigmon.New(runner, os.Kill)

	return sigRunner
}

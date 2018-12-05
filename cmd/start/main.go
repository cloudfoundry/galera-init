package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"os/exec"
	"syscall"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/galera-init/cluster_health_checker"
	"github.com/cloudfoundry/galera-init/config"
	"github.com/cloudfoundry/galera-init/db_helper"
	"github.com/cloudfoundry/galera-init/os_helper"
	"github.com/cloudfoundry/galera-init/start_manager"
	"github.com/cloudfoundry/galera-init/start_manager/node_starter"
	"github.com/cloudfoundry/galera-init/upgrader"
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

	cfg.Logger.Info("galera-init starting")

	startManager := managerSetup(cfg)
	err = startManager.BlockingExecute()

	if err != nil {
		cfg.Logger.Error("Mysqld return an error: ", err)

		switch err.(type) {
		case *exec.ExitError:
			cfg.Logger.Error("Mysqld daemon ungracefully exited because: ", err)
			ws, ok := err.(*exec.ExitError).Sys().(syscall.WaitStatus)
			if !ok {
				cfg.Logger.Error("Unable to determine exit status from error", err, lager.Data{
					"errType": err.(*exec.ExitError).Sys(),
				})
				os.Exit(1)
			}
			if ws.Signaled() {
				os.Exit(int(ws.Signal()))
			} else {
				os.Exit(ws.ExitStatus())
			}
		default:
			cfg.Logger.Error("Unhandled error in main(), exiting with 1: ", err)
			os.Exit(1)
		}
	}

	cfg.Logger.Info("galera-init shutting down!")
	os.Exit(0)
}

func writePidFile(cfg *config.Config) error {
	cfg.Logger.Info("Copying child pid to parent pid", lager.Data{
		"childPidfile": cfg.ChildPidFile,
		"pidfile":      cfg.PidFile,
	})
	pidAsByteArray, err := ioutil.ReadFile(cfg.ChildPidFile)
	if err != nil {
		panic(fmt.Sprintf("could not read pid file from %s", cfg.ChildPidFile))
	}
	return ioutil.WriteFile(cfg.PidFile, pidAsByteArray, 0644)
}

func deletePidFile(cfg *config.Config) error {
	cfg.Logger.Info("Deleting pidfile", lager.Data{
		"pidfile": cfg.PidFile,
	})
	return os.Remove(cfg.PidFile)
}

func managerSetup(cfg *config.Config) start_manager.StartManager {
	OsHelper := os_helper.NewImpl()

	DBHelper := db_helper.NewDBHelper(
		OsHelper,
		&cfg.Db,
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

	NodeStarter := node_starter.NewStarter(
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

	return NodeStartManager
}

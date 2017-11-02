package prestarter

import (
	//"github.com/cloudfoundry/mariadb_ctrl/os_helper"
	//"github.com/cloudfoundry/mariadb_ctrl/mariadb_helper"
	//"github.com/cloudfoundry/mariadb_ctrl/upgrader"
	//"github.com/cloudfoundry/mariadb_ctrl/start_manager/node_starter"
	//"code.cloudfoundry.org/lager"
	//"github.com/cloudfoundry/mariadb_ctrl/cluster_health_checker"
)


type PreStarter interface {
	PreStart() error
}

type preStarter struct {
	manager StartManager
	args    []string
}

func New(
	manager StartManager,
	args []string,
) PreStarter {
	return &preStarter{
		manager: manager,
		args: args,
	}
}

//func New(
//	manager StartManager,
//	args []string,
//) StartManager {
//	return &PreStarter{
//		manager: manager,
//		args: args,
//	}
//}
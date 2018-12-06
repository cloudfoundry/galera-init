package test_helpers

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/onsi/ginkgo"
	"io"
)

type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type ContainerOption func(*docker.CreateContainerOptions)

func RunExec(dockerClient *docker.Client, exec *docker.Exec) (*ExecResult, error) {
	var (
		stdout, stderr bytes.Buffer
	)

	err := dockerClient.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: io.MultiWriter(ginkgo.GinkgoWriter, &stdout),
		ErrorStream:  io.MultiWriter(ginkgo.GinkgoWriter, &stderr),
	})
	if err != nil {
		return nil, err
	}

	execInspect, err := dockerClient.InspectExec(exec.ID)
	if err != nil {
		return nil, err
	}

	return &ExecResult{
		ExitCode: execInspect.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

func CreateNetwork(dockerClient *docker.Client, name string) (*docker.Network, error) {
	return dockerClient.CreateNetwork(docker.CreateNetworkOptions{Name: name})
}

// FIXME: handle errors if network settings are not configured properly
func HostPort(containerPort docker.Port, container *docker.Container) string {
	return container.NetworkSettings.Ports[containerPort][0].HostPort
}

func PullImage(dockerClient *docker.Client, repository string) error {
	return dockerClient.PullImage(docker.PullImageOptions{
		Repository:   repository,
		OutputStream: ginkgo.GinkgoWriter,
	}, docker.AuthConfiguration{})
}

func RemoveNetwork(dockerClient *docker.Client, network *docker.Network) error {
	return dockerClient.RemoveNetwork(network.ID)
}

func RemoveContainer(dockerClient *docker.Client, container *docker.Container) error {
	if err := dockerClient.StopContainer(container.ID, 0); err != nil {
		return err
	}

	return dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
}

func RunContainer(dockerClient *docker.Client, containerName string, options ...ContainerOption) (*docker.Container, error) {
	createContainerOptions := docker.CreateContainerOptions{
		Name: containerName,
		Config: &docker.Config{
			ExposedPorts: map[docker.Port]struct{}{},
		},
		HostConfig: &docker.HostConfig{
			PublishAllPorts: true,
		},
	}

	for _, opt := range options {
		opt(&createContainerOptions)
	}

	container, err := dockerClient.CreateContainer(createContainerOptions)
	if err != nil {
		return container, err
	}

	if err := dockerClient.StartContainer(container.ID, nil); err != nil {
		return container, err
	}

	inspectedContainer, err := dockerClient.InspectContainer(container.ID)
	if err != nil {
		return container, err
	}

	return inspectedContainer, nil
}

func AddBinds(binds ...string) ContainerOption {
	return func(createOpts *docker.CreateContainerOptions) {
		createOpts.HostConfig.Binds = append(createOpts.HostConfig.Binds, binds...)
	}
}

func AddEnvVars(envVars ...string) ContainerOption {
	return func(createOpts *docker.CreateContainerOptions) {
		createOpts.Config.Env = append(createOpts.Config.Env, envVars...)
	}
}

func AddExposedPorts(ports ...docker.Port) ContainerOption {
	return func(createOpts *docker.CreateContainerOptions) {
		for _, port := range ports {
			createOpts.Config.ExposedPorts[port] = struct{}{}
		}
	}

}

func WithEntrypoint(cmd string) ContainerOption {
	return func(createOpts *docker.CreateContainerOptions) {
		createOpts.Config.Entrypoint = []string{cmd}
	}
}

func WithCmd(cmd ...string) ContainerOption {
	return func(createOpts *docker.CreateContainerOptions) {
		createOpts.Config.Cmd = cmd
	}
}

func WithImage(imageID string) ContainerOption {
	return func(createOpts *docker.CreateContainerOptions) {
		createOpts.Config.Image = imageID
	}
}

func WithNetwork(network *docker.Network) ContainerOption {
	return func(createOpts *docker.CreateContainerOptions) {
		createOpts.NetworkingConfig = &docker.NetworkingConfig{
			EndpointsConfig: map[string]*docker.EndpointConfig{
				network.Name: {NetworkID: network.ID},
			},
		}
	}
}

func ContainerDBConnection(container *docker.Container, port docker.Port) (*sql.DB, error) {
	hostPort := HostPort(port, container)
	dbURI := fmt.Sprintf("root@tcp(localhost:%s)/?multiStatements=true", hostPort)
	return sql.Open("mysql", dbURI)
}

func IsReadOnly(db *sql.DB) (bool, error) {
	var isReadOnly bool
	err := db.QueryRow("select @@global.read_only;").Scan(&isReadOnly)
	return isReadOnly, err
}

func SetReadOnly(db *sql.DB, val bool) error {
	_, err := db.Exec("set @@global.super_read_only=?", val)
	return err
}

func TerminateMySQLD(dockerClient *docker.Client, container *docker.Container) (*ExecResult, error) {
	exec, err := dockerClient.CreateExec(docker.CreateExecOptions{
		Container: container.ID,
		Cmd: []string{
			"bash",
			"-c",
			"kill -9 $(pidof mysqld)",
		},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, err
	}

	return RunExec(dockerClient, exec)
}

func DbSchemaExists(db *sql.DB, schemaName string) (bool, error) {
	query := fmt.Sprintf(
		`SELECT COUNT(*) = 1 FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = '%s'`,
		schemaName,
	)

	var count string
	err := db.QueryRow(query).Scan(&count)
	return count == "1", err
}

func DbTableExists(db *sql.DB, schemaName, tableName string) (bool, error) {
	query := fmt.Sprintf(
		`SELECT COUNT(*) = 1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'`,
		schemaName, tableName)

	var count string
	err := db.QueryRow(query).Scan(&count)
	return count == "1", err
}

type MySQLEvent struct {
	DB            string
	Name          string
	Type          string
	IntervalValue string
	IntervalField string
	Status        string
}

func ShowEvent(db *sql.DB, schemaName, eventName string) (MySQLEvent, error) {
	sql := fmt.Sprintf(`SELECT EVENT_SCHEMA, EVENT_NAME, EVENT_TYPE, INTERVAL_VALUE, INTERVAL_FIELD, STATUS FROM information_schema.events WHERE EVENT_SCHEMA = '%s' AND EVENT_NAME = '%s'`,
		schemaName, eventName)
	var event MySQLEvent
	err := db.QueryRow(sql).Scan(
		&event.DB,
		&event.Name,
		&event.Type,
		&event.IntervalValue,
		&event.IntervalField,
		&event.Status,
	)
	return event, err
}

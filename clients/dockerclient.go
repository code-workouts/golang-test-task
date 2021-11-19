package clients

import (
	"bufio"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang-test-task/envconfig"
	"log"
	"strings"
	"time"
)

type ContainerConfig struct {
	Image string
	Cmd   []string
}

func NewContainerConfig(image string, bashCommand string) *ContainerConfig {
	return &ContainerConfig{
		Image: image,
		Cmd:   []string{"bin/bash", "-c", bashCommand},
	}
}

func (containerConfig *ContainerConfig) RunContainerAndStreamLogs(ctx context.Context, logStream chan<- string) {
	// Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// Docker image pull
	reader, err := cli.ImagePull(ctx, containerConfig.Image, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	//io.Copy(os.Stdout, reader)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		log.Println(scanner.Text())
	}

	// Container configuration
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        containerConfig.Image,
		Cmd:          containerConfig.Cmd,
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	// Starting container
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	//Stream container logs
	streamContainerLogs(ctx, cli, resp.ID, logStream)
}

func streamContainerLogs(ctx context.Context, cli *client.Client, containerId string, logStream chan<- string) {
	reader, err := cli.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		panic(err)
	}

	defer func() {
		reader.Close()
		close(logStream)
	}()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		msg := getBatchOutput(scanner, envconfig.LogBatchDuration)
		logStream <- msg
		log.Printf("===========Streaming log Patch:=========== \n%s\n", msg)
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil && !strings.Contains(err.Error(), context.Canceled.Error()) {
			panic(err)
		} else {
			log.Println("Container wait:", context.Canceled)
		}
	case <-statusCh:
	}

	select {
	case <-ctx.Done():
		log.Println("Log Stream:", ctx.Err())
		containerCleanup(cli, containerId)
	}
}

func containerCleanup(cli *client.Client, containerId string) {
	timeout := time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout*2)
	defer cancel()

	log.Printf("%s: Executing container stop....\n", context.Canceled)
	err := cli.ContainerStop(ctx, containerId, &timeout)
	if err != nil {
		log.Println("Error found on Container stop: ", err)
	}

	log.Printf("%s: Executing container remove....\n", context.Canceled)
	err = cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		log.Println("Error found on Container remove: ", err)
	}

	select {
	case <-ctx.Done():
		log.Println("Container cleanup:", ctx.Err())
	}
}

func getBatchOutput(scanner *bufio.Scanner, until time.Duration) string {
	var b strings.Builder
	timeNow := time.Now()

	msg := scanner.Text()
	b.WriteString(msg + "\n")
	for time.Since(timeNow) <= until && scanner.Scan() {
		msg := scanner.Text()
		b.WriteString(msg + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

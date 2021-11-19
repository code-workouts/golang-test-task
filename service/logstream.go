package service

import (
	"context"
	"golang-test-task/clients"
)

func StreamDockerLogsToCloudWatch(ctx context.Context,
	dockerClient *clients.ContainerConfig,
	awsCwlClient *clients.AwsCloudWatchLogs,
	cloudwatchGroup *string,
	cloudwatchStream *string) {

	// Output stream
	logStream := make(chan string, 20)
	logPushStatus := make(chan bool)

	// Run Docker container and stream the output
	go dockerClient.RunContainerAndStreamLogs(ctx, logStream)

	// Push log events
	go awsCwlClient.PutLogEvents(*cloudwatchGroup, *cloudwatchStream, logStream, logPushStatus)

	<-logPushStatus
}

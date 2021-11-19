package main

import (
	"context"
	"flag"
	"golang-test-task/clients"
	"golang-test-task/service"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	dockerImage        *string
	bashCommand        *string
	cloudwatchGroup    *string
	cloudwatchStream   *string
	awsAccessKeyId     *string
	awsSecretAccessKey *string
	awsRegion          *string
)

func main() {
	parseCmdLineArgs(false)

	ctx, cancel := context.WithCancel(context.Background())
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL)

	defer func() {
		signal.Stop(sigc)
		cancel()
	}()

	go func() {
		select {
		case <-sigc:
			cancel()
		}
	}()

	// Initiate AwsCloudWatchLogs
	awsCwlClient := clients.NewAwsCloudWatchLogs(awsRegion, awsAccessKeyId, awsSecretAccessKey)

	// Initiate Docker client
	dockerClient := clients.NewContainerConfig(*dockerImage, *bashCommand)

	// Stream Docker log to Cloud Watch
	service.StreamDockerLogsToCloudWatch(ctx, dockerClient, awsCwlClient, cloudwatchGroup, cloudwatchStream)
}

func parseCmdLineArgs(printArgs bool) {
	dockerImage = flag.String("docker-image", "", "Docker image to deploy")
	bashCommand = flag.String("bash-command", "", "Bash command to run in container")
	cloudwatchGroup = flag.String("cloudwatch-group", "", "AWS cloud watch group")
	cloudwatchStream = flag.String("cloudwatchstream", "", "AWS cloud watch stream")
	awsAccessKeyId = flag.String("aws-access-key-id", "", "AWS access key id")
	awsSecretAccessKey = flag.String("aws-secret-access-key", "", "AWS access key")
	awsRegion = flag.String("aws-region", "", "AWS Region")
	flag.Parse()

	if len(flag.Args()) > 0 {
		log.Printf("Unrecognized commands - %s\n", flag.Args())
		os.Exit(1)
	}

	checkRequiredArgs()

	if printArgs {
		log.Printf("dockerImage: %s", *dockerImage)
		log.Printf("bashCommand: %s", *bashCommand)
		log.Printf("cloudwatchGroup: %s", *cloudwatchGroup)
		log.Printf("cloudwatchStream: %s", *cloudwatchStream)
		log.Printf("awsAccessKeyId: %s", *awsAccessKeyId)
		log.Printf("awsSecretAccessKey: %s", *awsSecretAccessKey)
		log.Printf("awsRegion: %s", *awsRegion)
	}
}

func checkRequiredArgs() {
	if *dockerImage == "" {
		log.Printf("Missing required argument 'docker-image'\n")
		flag.Usage()
		os.Exit(1)
	}

	if *bashCommand == "" {
		log.Printf("Missing required argument 'bash-command'\n")
		flag.Usage()
		os.Exit(1)
	}

	if *cloudwatchGroup == "" {
		log.Printf("Missing required argument 'cloudwatch-group'\n")
		flag.Usage()
		os.Exit(1)
	}

	if *cloudwatchStream == "" {
		log.Printf("Missing required argument 'cloudwatchstream'\n")
		flag.Usage()
		os.Exit(1)
	}

	if *awsAccessKeyId == "" {
		log.Printf("Missing required argument 'aws-access-key-id'\n")
		flag.Usage()
		os.Exit(1)
	}

	if *awsSecretAccessKey == "" {
		log.Printf("Missing required argument 'aws-secret-access-key'\n")
		flag.Usage()
		os.Exit(1)
	}

	if *awsRegion == "" {
		log.Printf("Missing required argument 'aws-region'\n")
		flag.Usage()
		os.Exit(1)
	}
}

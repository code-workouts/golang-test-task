package clients

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"golang-test-task/envconfig"
	"log"
	"regexp"
	"strings"
	"time"
)

type AwsSession struct {
	// Regions and Endpoints.
	Region *string

	// AWS Access key ID
	AccessKeyID *string

	// AWS Secret Access Key
	SecretAccessKey *string
}

func (awsSession *AwsSession) NewAwsSession() *session.Session {
	session, err := session.NewSession(&aws.Config{
		Region:      awsSession.Region,
		Credentials: credentials.NewStaticCredentials(*awsSession.AccessKeyID, *awsSession.SecretAccessKey, ""),
	})
	if err != nil {
		panic(err)
	}

	return session
}

type AwsCloudWatchLogs struct {
	CloudWatchLogs *cloudwatchlogs.CloudWatchLogs
}

func NewAwsCloudWatchLogs(awsRegion *string, awsAccessKeyId *string, awsSecretAccessKey *string) *AwsCloudWatchLogs {
	awsSession := AwsSession{
		Region:          awsRegion,
		AccessKeyID:     awsAccessKeyId,
		SecretAccessKey: awsSecretAccessKey,
	}
	session := awsSession.NewAwsSession()

	return &AwsCloudWatchLogs{CloudWatchLogs: cloudwatchlogs.New(session)}
}

func (awscwl *AwsCloudWatchLogs) EnsureLogGroupExists(name string) error {
	resp, err := awscwl.CloudWatchLogs.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{})
	if err != nil {
		return err
	}

	for _, logGroup := range resp.LogGroups {
		if *logGroup.LogGroupName == name {
			return nil
		}
	}

	_, err = awscwl.CloudWatchLogs.CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: &name,
	})
	if err != nil {
		return err
	}

	_, err = awscwl.CloudWatchLogs.PutRetentionPolicy(&cloudwatchlogs.PutRetentionPolicyInput{
		RetentionInDays: aws.Int64(14),
		LogGroupName:    &name,
	})

	return err
}

func (awscwl *AwsCloudWatchLogs) EnsureLogStreamExists(logStreamName string, logGroupName string) error {
	resp, err := awscwl.CloudWatchLogs.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{LogGroupName: &logGroupName})
	if err != nil {
		return err
	}

	for _, logStream := range resp.LogStreams {
		if *logStream.LogStreamName == logStreamName {
			return nil
		}
	}

	_, err = awscwl.CloudWatchLogs.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  &logGroupName,
		LogStreamName: &logStreamName,
	})

	return err
}

func (awscwl *AwsCloudWatchLogs) PutLogEvents(logGroupName string, logStreamName string, logStream <-chan string, logPushStatus chan<- bool) {
	err := awscwl.EnsureLogGroupExists(logGroupName)
	if err != nil {
		panic(err)
	}

	err = awscwl.EnsureLogStreamExists(logStreamName, logGroupName)
	if err != nil {
		panic(err)
	}

	//Log events
	var logEvents []*cloudwatchlogs.InputLogEvent
	for {
		var more bool
		var msg string
		var nextSequenceToken *string

		for i := 1; i <= envconfig.LogEventsCount; i++ {
			msg, more = <-logStream
			if more {
				logEvents = append(logEvents, &cloudwatchlogs.InputLogEvent{
					Message:   aws.String(msg),
					Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
				})
				time.Sleep(time.Second * 2)
			} else {
				break
			}
		}

		if len(logEvents) > 0 {
			var putLogEventsOutput *cloudwatchlogs.PutLogEventsOutput
			putLogEventsOutput, err = awscwl.CloudWatchLogs.PutLogEvents(&cloudwatchlogs.PutLogEventsInput{
				LogEvents:     logEvents,
				LogGroupName:  &logGroupName,
				LogStreamName: &logStreamName,
				SequenceToken: nextSequenceToken,
			})
			if err != nil {
				if strings.HasPrefix(err.Error(), cloudwatchlogs.ErrCodeInvalidSequenceTokenException) {
					r := regexp.MustCompile(`ExpectedSequenceToken: "(.*)"`)
					seqToken := r.FindStringSubmatch(err.Error())[1]
					log.Printf("Caught %s retrying again with ExpectedSequenceToken %s\n", cloudwatchlogs.ErrCodeInvalidSequenceTokenException, seqToken)
					nextSequenceToken = &seqToken
					putLogEventsOutput, err = awscwl.CloudWatchLogs.PutLogEvents(&cloudwatchlogs.PutLogEventsInput{
						LogEvents:     logEvents,
						LogGroupName:  &logGroupName,
						LogStreamName: &logStreamName,
						SequenceToken: nextSequenceToken,
					})
				}
				if err != nil {
					panic(err)
				}
			}
			nextSequenceToken = putLogEventsOutput.NextSequenceToken
		}

		if !more {
			break
		}
	}

	log.Println("Logged all events to cloud watch")
	logPushStatus <- true
}

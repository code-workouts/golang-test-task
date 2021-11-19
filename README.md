# golang-test-task

Run a docker container with a command as a serverless and upstream the logs to aws cloudwatch with given AWS credentials.

## Dependencies
```bash
go mod tidy
```

## Build
```bash
go build .
```

## Run
```bash
./golang-test-task --docker-image python \
--bash-command 'pip install pip -U && pip install tqdm && python -c "exec(\"import time\ncounter = 0\nwhile True:\n  print(counter)\n  counter= counter + 1\n  time.sleep(0.1)\")"' \
--cloudwatch-group golang-test-task-group-1 \
--cloudwatchstream golang-test-task-group-2 \
--aws-access-key-id '<AWS_ACCESS_KEY_ID>' \
--aws-secret-access-key '<AWS_SECRET_ACCESS_KEY>' \
--aws-region 'ap-south-1'
```

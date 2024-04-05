[![Build Status](https://travis-ci.org/kayac/Gunfish.svg?branch=master)](https://travis-ci.org/kayac/Gunfish)

# Gunfish

APNs and FCM provider server on HTTP/2.

* Gunfish provides the interface as the APNs / FCM provider server.

## Overview

![overviews 1](https://cloud.githubusercontent.com/assets/13774847/14844813/17035232-0c95-11e6-8307-1d8340978bb7.png)

[Gunfish slides](http://slides.com/takuyayoshimura-tkyshm/deck-1/fullscreen)

[Gunfish slides (jp)](http://slides.com/takuyayoshimura-tkyshm/deck/fullscreen)

## Install

### Binary

Download the latest binary from [releases](https://github.com/kayac/Gunfish/releases)

### Docker images

[DockerHub](https://hub.docker.com/r/kayac/gunfish/)

[GitHub Packages](https://github.com/kayac/Gunfish/pkgs/container/gunfish)

### Homebrew

```console
$ brew tap kayac/tap
$ brew install gunfish
```

## Quick Started

```bash
$ gunfish -c ./config/gunfish.toml -E production
```

### Commandline Options

option              | required | description
------------------- |----------|------------------------------------------------------------------------------------------------------------------
-port               | Optional | Port number of Gunfish provider server. Default is `8003`.
-environment, -E    | Optional | Default value is `production`.
-conf, -c           | Optional | Please specify this option if you want to change `toml` config file path. (default: `/etc/gunfish/config.toml`.)
-log-level          | Optional | Set the log level as 'warn', 'info', or 'debug'.
-log-format         | Optional | Supports `json` or `ltsv` log formats.
-enable-pprof       | Optional | You can set the flag of pprof debug port open.
-output-hook-stdout | Optional | Merge stdout of hook command to gunfish's stdout.
-output-hook-stderr | Optional | Merge stderr of hook command to gunfish's stderr.

## API

### POST /push/apns

To delivery remote notifications via APNS to user's devices.

param | description
--- | ---
Array | Array of JSON dictionary includes 'token' and 'payload' properties

payload param | description
--- | ---
token | Published token from APNS to user's remote device
payload | APNS notification payload

Post JSON example:
```json
[
  {
    "payload": {
      "aps": {
        "alert": "test notification",
        "sound": "default"
      },
      "option1": "foo",
      "option2": "bar"
    },
    "token": "apns device token",
    "header": {
      "apns-id": "your apns id",
      "apns-topic": "your app bundle id",
      "apns-push-type": "alert"
    }
  }
]
```

Response example:
```json
{"result": "ok"}
```

### POST /push/fcm **Deprecated**

This API has been deleted at v0.6.0. Use `/push/fcm/v1` instead.

See also https://firebase.google.com/docs/cloud-messaging/migrate-v1 .

### POST /push/fcm/v1

To delivery remote notifications via FCM v1 API to user's devices.

Post body format is equal to it for FCM v1 origin server.

example:
```json
{
  "message": {
    "notification": {
      "title": "message_title",
      "body": "message_body",
      "image": "https://example.com/notification.png"
    },
    "data": {
      "sample_key": "sample key",
      "message": "sample message"
    },
    "token": "InstanceIDTokenForDevice"
  }
}
```

Response example:
```json
{"result": "ok"}
```

FCM v1 endpoint allows multiple payloads in a single request body. You can build request body simply concat multiple JSON payloads. Gunfish sends for each that payloads to FCM server. Limitation: Max count of payloads in a request body is 500.

### GET /stats/app

```json
{
  "pid": 57843,
  "debug_port": 0,
  "uptime": 384,
  "start_at": 1492476864,
  "su_at": 0,
  "period": 309,
  "retry_after": 10,
  "workers": 8,
  "queue_size": 0,
  "retry_queue_size": 0,
  "workers_queue_size": 0,
  "cmdq_queue_size": 0,
  "retry_count": 0,
  "req_count": 0,
  "sent_count": 0,
  "err_count": 0,
  "certificate_not_after": "2027-04-16T00:53:53Z",
  "certificate_expire_until": 315359584
}
```

To get the status of APNS proveder server.

stats type | description
--- | ---
pid | PID
debug\_port | pprof port number
uptime | uptime
workers | number of workers
start\_at | The time of started
queue\_size | queue size of requests
retry\_queue\_size | queue size for resending notification
workers\_queue\_size | summary of worker's queue size
command\_queue\_size | error hook command queue size
retry\_count | summary of retry count
request\_count | request count to gunfish
err\_count | count of recieving error response
sent\_count | count of sending notification
certificate\_not\_after | certificates minimum expiration date for APNs
certificate\_expire\_until | certificates minimum expiration untile (sec)

### GET /stats/profile

To get the status of go application.

See detail properties that url: (https://github.com/fukata/golang-stats-api-handler).

## Configuration
The Gunfish configuration file is a TOML file that Gunfish server uses to configure itself.
That configuration file should be located at `/etc/gunfish.toml`, and is required to start.
Here is an example configuration:

```toml
[provider]
port = 8003
worker_num = 8
queue_size = 2000
max_request_size = 1000
max_connections = 2000
error_hook = "echo -e 'Hello Gunfish at error hook!'"

[apns]
key_file = "/path/to/server.key"
cert_file = "/path/to/server.crt"
kid = "kid"
team_id = "team_id"

[fcm_v1]
google_application_credentials = "/path/to/credentials.json"
```

### [provider] section

This section is for Gunfish server configuration.

Parameter        | Requirement | Description
---------------- | ------ | --------------------------------------------------------------------------------------
port             |optional| Listen port number.
worker_num       |optional| Number of Gunfish owns http clients.
queue_size       |optional| Limit number of posted JSON from the developer application.
max_request_size |optional| Limit size of Posted JSON array.
max_connections  |optional| Max connections
error_hook       |optional| Error hook command. This command runs when Gunfish catches an error response.

### [apns] section

This section is for APNs provider configuration.
If you don't need to APNs provider, you can skip this section.

Parameter        | Requirement | Description
---------------- | ------ | --------------------------------------------------------------------------------------
key_file         |required| The key file path.
cert_file        |optional| The cert file path.
kid              |optional| kid for APNs provider authentication token.
team_id          |optional| team id for APNs provider authentication token.

### [fcm_v1] section

This section is for FCM v1 provider configuration.
If you don't need to FCM v1 provider, you can skip this section.

Parameter        | Requirement | Description
---------------- | ------ | --------------------------------------------------------------------------------------
google_application_credentials |required| The path to the Google Cloud Platform service account key file.

## Error Hook

Error hook command can get an each error response with JSON format by STDIN.

for example JSON structure: (>= v0.2.x)
```json5
// APNs
{
  "provider": "apns",
  "apns-id": "123e4567-e89b-12d3-a456-42665544000",
  "status": 400,
  "token": "9fe817acbcef8173fb134d8a80123cba243c8376af83db8caf310daab1f23003",
  "reason": "MissingTopic"
}
```

```json5
// FCM v1
{
  "provider": "fcmv1",
  "status": 400,
  "token": "testToken",
  "error": {
    "status": "INVALID_ARGUMENT",
    "message": "The registration token is not a valid FCM registration token"
  }
}
```

## Graceful Restart
Gunfish supports graceful restarting based on `Start Server`. So, you should start on `start_server` command if you want graceful to restart.

```bash
### install start_server
$ go get github.com/lestrrat/go-server-starter/cmd/start_server

### Starts Gunfish with start_server
$ start_server --port 38003 --pid-file gunfish.pid -- ./gunfish -c conf/gunfish.toml
```

### Test

```
$ make test
```

The following tools are useful to send requests to gunfish for test the following.
- gunfish-cli (send push notification to Gunfish for test)
- apnsmock (APNs mock server)

```
$ make tools/gunfish-cli
$ make tools/apnsmock
```

- send a request example with gunfish-cli
```
$ ./gunfish-cli -type apns -count 1 -json-file some.json -verbose
$ ./gunfish-cli -type apns -count 1 -token <device token> -apns-topic <your topic> -options key1=val1,key2=val2 -verbose
```

- start apnsmock server
```
$ ./apnsmock -cert-file ./test/server.crt -key-file ./test/server.key -verbose
```

### Benchmark

Gunfish repository includes Lua script for the benchmark. You can use wrk command with `err_and_success.lua` script.

```
$ make tools/apnsmock
$ ./apnsmock -cert-file ./test/server.crt -key-file ./test/server.key -verbosea &
$ ./gunfish -c test/gunfish_test.toml -E test
$ wrk2 -t2 -c20 -s bench/scripts/err_and_success.lua -L -R100 http://localhost:38103
```

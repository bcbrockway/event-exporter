# Event-exporter

Export Kubernetes' events to Elasticsearch/TCP/HTTP Endpoint.

Inspired by https://github.com/GoogleCloudPlatform/k8s-stackdriver/tree/master/event-exporter .

## Changes from fork

- Added TCP sink
- Fixed various typos
- Added `dep` and updated `Makefile` and `Dockerfile`

# Build and run

`git clone` to your `$GOPATH`

```
make build
./bin/event-exporter
```

# Examples

```
# Use ncat to listen on a socket
>> ncat -lk 9000

# Use the tcp-sink (default 127.0.0.1:9000). Note, this also uses $HOME/.kube/config by default
>> ./bin/event-exporter -v 10 -logtostderr -sink "tcp" -prometheus-endpoint ":8000"
```

# How to config

## Event exporter options:
   
```
Usage of ./bin/event-exporter:
  -alsologtostderr
    	log to standard error as well as files
  -auth string
    	Http auth method: basic or token (default "token")
  -elasticsearch-server string
    	Elasticsearch endpoint (default "http://elasticsearch:9200/")
  -http-endpoint string
    	Http endpoint
  -incluster
    	use in cluster configuration.
  -kubeconfig string
    	path to kubeconfig (if not in running inside a cluster) (default "/home/nb/.kube/config")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -password string
    	Nassword for http basic auth
  -prometheus-endpoint string
    	Endpoint on which to expose Prometheus http handler (default ":80")
  -resync-period duration
    	Reflector resync period (default 1m0s)
  -sink string
    	Sink type to save the exported events: elasticsearch/http (default "elasticsearch")
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -tcp-endpoint string
    	TCP endpoint (default "127.0.0.1:9000")
  -token string
    	Token header and value for http token auth
  -username string
    	Username for http basic auth
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```

## Sinks

### ElasticSearch

Output results directly to an ElasticSearch stack.

### TCP

Output results directly to a TCP socket (useful for filebeat/etc)

### HTTP

Output results directly to an HTTP endpoint (useful for filebeat/etc)


# Deploy on kubernetes

```
apiVersion: v1
kind: Service
metadata:
  name: event-exporter
  namespace: kube-system
spec:
  ports:
  - port: 80
    targetPort: 80
  selector:
    run: event-exporter
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    run: event-exporter
  name: event-exporter
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      run: event-exporter
  template:
    metadata:
      labels:
        run: event-exporter
    spec:
      containers:
      - image: mintel/event-exporter
        ports:
        - containerPort: 80 
        imagePullPolicy: Always
        name: event-exporter
        command: ["/event-exporter"]
        args: ["-v", "4"]
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      terminationGracePeriodSeconds: 30
```

/*
Copyright 2017 Google Inc.
Copyright 2018 Alauda Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nabadger/event-exporter/sinks"
)

const (
	sinkNameElasticSearch = "elasticsearch"
	sinkNameHTTP          = "http"
	sinkNameTCP		  = "tcp"
)

var (
	inCluster          = flag.Bool("incluster", false, "use in cluster configuration.")
	kubeconfig         = flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "path to kubeconfig (if not in running inside a cluster)")
	resyncPeriod       = flag.Duration("resync-period", 1*time.Minute, "Reflector resync period")
	prometheusEndpoint = flag.String("prometheus-endpoint", ":80", "Endpoint on which to "+
		"expose Prometheus http handler")
	sinkName              = flag.String("sink", sinkNameElasticSearch, "Sink type to save the exported events: elasticsearch/http")
	elasticsearchEndpoint = flag.String("elasticsearch-server", "http://elasticsearch:9200/", "Elasticsearch endpoint")

	// for http sink
	httpEndpoint = flag.String("http-endpoint", "", "Http endpoint")
	httpAuth     = flag.String("auth", "token", "Http auth method: basic or token")
	httpToken    = flag.String("token", "", "Token header and value for http token auth")
	httpUsername = flag.String("username", "", "Username for http basic auth")
	httpPassword = flag.String("password", "", "Nassword for http basic auth")

	// for tcp sink
	tcpEndpoint = flag.String("tcp-endpoint", "127.0.0.1:9000", "TCP endpoint")
)

func newSystemStopChannel() chan struct{} {
	ch := make(chan struct{})
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		sig := <-c
		glog.Infof("Recieved signal %s, terminating", sig.String())

		ch <- struct{}{}
	}()

	return ch
}

func newKubernetesClient() (kubernetes.Interface, error) {
	var err error
	var config *rest.Config
	if *kubeconfig != "" && !*inCluster {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %v", err)
	}
	return client, err
}

func main() {
	flag.Set("logtostderr", "true")
	defer glog.Flush()
	flag.Parse()
	var outSink sinks.Sink
	var err error
	var k8sClient kubernetes.Interface

	if *sinkName == sinkNameElasticSearch {
		config := sinks.DefaultElasticSearchConf()
		config.Endpoint = *elasticsearchEndpoint
		outSink, err = sinks.NewElasticSearchSink(config)
		if err != nil {
			glog.Fatalf("Failed to initialize elasticsearch output: %v", err)
		}
	} else if *sinkName == sinkNameHTTP {
		config := sinks.DefaultHTTPConf()
		config.Endpoint = httpEndpoint
		config.Auth = httpAuth
		config.Token = httpToken
		config.Username = httpUsername
		config.Password = httpPassword
		outSink, err = sinks.NewHTTPSink(config)
		if err != nil {
			glog.Fatalf("Failed to initialize http output: %v", err)
		}
	} else if *sinkName == sinkNameTCP {
		config := sinks.DefaultTCPConf()
		config.Endpoint = tcpEndpoint
		outSink, err = sinks.NewTCPSink(config)
		if err != nil {
			glog.Fatalf("Failed to initiaize tcp output: %v", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(-1)
	}

	k8sClient, err = newKubernetesClient()
	if err != nil {
		glog.Fatalf("Failed to initialize kubernetes client: %v", err)
	}

	eventExporter := newEventExporter(k8sClient, outSink, *resyncPeriod)

	// Expose the Prometheus http endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		glog.Fatalf("Prometheus monitoring failed: %v", http.ListenAndServe(*prometheusEndpoint, nil))
	}()

	stopCh := newSystemStopChannel()
	eventExporter.Run(stopCh)
}

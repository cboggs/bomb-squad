package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Fresh-Tracks/bomb-squad/config"
	configmap "github.com/Fresh-Tracks/bomb-squad/k8s/configmap"
	"github.com/Fresh-Tracks/bomb-squad/patrol"
	"github.com/Fresh-Tracks/bomb-squad/prom"
	"github.com/Fresh-Tracks/bomb-squad/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	version          = "undefined"
	promVersion      = "undefined"
	promRulesVersion = "undefined"
	inK8s            = flag.Bool("k8s", false, "Whether bomb-squad is being deployed in a Kubernetes cluster")
	k8sNamespace     = flag.String("k8s-namespace", "default", "Kubernetes namespace holding Prometheus ConfigMap")
	k8sConfigMapName = flag.String("k8s-configmap", "prometheus", "Name of the Kubernetes ConfigMap holding Prometheus configuration")
	metricsPort      = flag.Int("metrics-port", 8080, "Port on which to listen for metric scrapes")
	promURL          = flag.String("prom-url", "http://localhost:9090", "Prometheus URL to query")
	cmName           = flag.String("configmap-name", "prometheus", "Name of the Prometheus ConfigMap")
	cmKey            = flag.String("configmap-prometheus-key", "prometheus.yml", "The key in the ConfigMap that holds the main Prometheus config")
	getVersion       = flag.Bool("version", false, "return version information and exit")
	versionGauge     = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "bomb_squad",
			Name:      "details",
			Help:      "Static series that tracks the current versions of all the things in Bomb Squad",
			ConstLabels: map[string]string{
				"version":                  version,
				"prometheus_version":       promVersion,
				"prometheus_rules_version": promRulesVersion,
			},
		},
	)
	k8sClientSet     kubernetes.Interface
	promConfigurator config.Configurator
	bsConfigurator   config.Configurator
)

func init() {
	prometheus.MustRegister(versionGauge)
	prometheus.MustRegister(patrol.ExplodingLabelGauge)
}

func bootstrap(c config.Configurator) {
	// TODO: Don't do this file write if the file already exists, but DO write the file
	// if it's not present on disk but still present in the ConfigMap
	b, err := ioutil.ReadFile("/etc/bomb-squad/rules.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("/etc/config/bomb-squad/rules.yaml", b, 0644)

	prom.AppendRuleFile("/etc/config/bomb-squad/rules.yaml", c)
}

func main() {
	flag.Parse()
	if *getVersion {
		out := ""
		for k, v := range map[string]string{
			"version":          version,
			"prometheus":       promVersion,
			"prometheus-rules": promRulesVersion,
		} {
			out = out + fmt.Sprintf("%s: %s\n", k, v)
		}
		log.Fatal(out)
	}

	if *inK8s {
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}

		k8sClientSet, err = kubernetes.NewForConfig(inClusterConfig)
		if err != nil {
			log.Fatal(err)
		}

		promConfigurator = configmap.NewConfigMapWrapper(k8sClientSet, *k8sNamespace, *k8sConfigMapName)
		bsConfigurator = configmap.NewConfigMapWrapper(k8sClientSet, *k8sNamespace, *k8sConfigMapName)
	}

	promurl, err := url.Parse(*promURL)
	if err != nil {
		log.Fatalf("could not parse prometheus url: %s", err)
	}
	log.Printf("PROMURL: %+v\n", promurl)

	httpClient, err := util.HttpClient()
	if err != nil {
		log.Fatalf("could not create http client: %s", err)
	}

	p := patrol.Patrol{
		PromURL:           promurl,
		Interval:          5 * time.Second,
		HighCardN:         5,
		HighCardThreshold: 100,
		HTTPClient:        httpClient,
		PromConfigurator:  promConfigurator,
		BSConfigurator:    bsConfigurator,
	}

	if len(os.Args) > 1 {
		cmd := os.Args[1]
		if cmd == "list" {
			fmt.Println("Suppressed Labels (metricName.labelName):")
			config.ListSuppressedMetrics(p.BSConfigurator)
			os.Exit(0)
		}

		if cmd == "unsilence" {
			label := os.Args[2]
			fmt.Printf("Removing silence rule for suppressed label: %s\n", label)
			config.RemoveSilence(label, p.PromConfigurator, p.BSConfigurator)
			os.Exit(0)
		}
	}

	if *inK8s {
		bootstrap(p.PromConfigurator)
	}
	go p.Run()

	mux := http.DefaultServeMux
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/metrics/reset", patrol.MetricResetHandler())
	versionGauge.Set(1.0)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *metricsPort),
		Handler: mux,
	}

	fmt.Println("Welcome to bomb-squad")
	log.Println("serving prometheus endpoints on port 8080")
	log.Fatal(server.ListenAndServe())
}

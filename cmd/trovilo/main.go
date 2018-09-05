package main

import (
	"context"

	"github.com/alecthomas/kingpin"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/inovex/trovilo/client"
	"github.com/inovex/trovilo/job"
	"github.com/inovex/trovilo/logging"
	"github.com/sirupsen/logrus"
)

var build string

func main() {
	// Prepare cmd line parser
	var configFile = kingpin.Flag("config", "YAML configuration file.").Required().ExistingFile()
	var kubeConfigFile = kingpin.Flag("kubeconfig", "Optional kubectl configuration file. If undefined we expect trovilo is running in a pod.").ExistingFile()
	var logLevel = kingpin.Flag("log-level", "Specify log level (debug, info, warn, error).").Default("info").String()
	var logJSON = kingpin.Flag("log-json", "Enable JSON-formatted logging on STDOUT.").Bool()

	kingpin.CommandLine.Help = "Trovilo collects and prepares files from Kubernetes ConfigMaps for Prometheus & friends"
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version(build)
	kingpin.CommandLine.VersionFlag.Short('v')
	kingpin.Parse()

	// Prepare logging
	log := logrus.New()
	logging.SetupLogging(log, *logJSON, *logLevel)

	// Prepare app configuration
	troviloJob, err := job.GetJob(log, *configFile)
	if err != nil {
		log.WithError(err).Fatal("Error parsing config file")
	}
	log.WithFields(logrus.Fields{"config": troviloJob}).Debug("Successfully loaded configuration")

	// Prepare Kubernetes client
	client, err := client.GetClient(*kubeConfigFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to load Kubernetes client")
	}
	log.Debug("Successfully loaded Kubernetes client")

	// TODO fixme
	log.Debug("Testing Kubernetes connectivity by fetching list of nodes..")
	if err := client.List(context.Background(), "", new(corev1.NodeList)); err != nil {
		log.WithError(err).Fatal("Failed to test Kubernetes connectivity")
	}
	log.Debug("Successfully tested Kubernetes connectivity")

	troviloJob.Initialize(log, client)
	troviloJob.Watch()
}

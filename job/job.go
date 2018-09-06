package job

import (
	"context"
	"io/ioutil"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

type VerifyStepCmd []string

type VerifyStep struct {
	Name string        `yaml:"name"`
	Cmd  VerifyStepCmd `yaml:"cmd"`
}

type PostDeployActionCmd []string

type PostDeployAction struct {
	Name string              `yaml:"name"`
	Cmd  PostDeployActionCmd `yaml:"cmd"`
}

type Job struct {
	Name       string             `yaml:"name"`
	Namespace  string             `yaml:"namespace"`
	Selector   map[string]string  `yaml:"selector"`
	Verify     []VerifyStep       `yaml:"verify"`
	TargetDir  string             `yaml:"target-dir"`
	Flatten    bool               `yaml:"flatten"`
	PostDeploy []PostDeployAction `yaml:"post-deploy"`
	watcher    *k8s.Watcher
	logger     *logrus.Logger
	client     *k8s.Client
}

func (job *Job) Initialize(logger *logrus.Logger, client *k8s.Client) {
	if logger != nil {
		job.logger = logger
	}

	job.logger.Infof("Configure job: %s", job.Name)
	selector := new(k8s.LabelSelector)

	for key, value := range job.Selector {
		selector.Eq(key, value)
	}

	watcher, err := client.Watch(context.Background(), job.Namespace, new(corev1.ConfigMap), selector.Selector())
	if err != nil {
		job.logger.WithError(err).Fatalf("Failed to load Kubernetes ConfigMap watcher for %s", job.Name)
	}

	job.watcher = watcher
}

func (job *Job) Watch() {
	defer job.watcher.Close()

	for {
		cm := new(corev1.ConfigMap)
		eventType, err := job.watcher.Next(cm)
		if err != nil {
			// TODO: can we test here for a specific type?
			// ensure watcher is closed
			job.watcher.Close()
			// recreate watcher
			job.Initialize(nil, job.client)
			//job.logger.WithError(err).Fatal("Kubernetes ConfigMap watcher encountered an error. Exit..")
		}

		logEntryBase := job.logger.WithFields(logrus.Fields{
			"job":       job.Name,
			"configmap": *cm.Metadata.Name,
			"namespace": *cm.Metadata.Namespace,
		})
		/*logEntryWithSelectors := logEntryBase.WithFields(logrus.Fields{
			"actualLabels":   cm.Metadata.Labels,
			"expectedLabels": job.Selector,
			"eventType":      eventType,
		})*/
		// Check whether ConfigMap matches to our expected labels
		if eventType == "DELETED" && IsCMAlreadyRegistered(cm, job.TargetDir, job.Flatten) {
			logEntryBase.Info("ConfigMap has been deleted from namepace, thus removing in target directory too")
			removedFiles, err := RemoveCMfromTargetDir(cm, job.TargetDir, job.Flatten)

			logEntry := logEntryBase.WithField("removedFiles", removedFiles)
			if err != nil {
				logEntry.WithError(err).Fatal("Failed to delete ConfigMap from namepace")
			} else {
				logEntry.Info("Successfully deleted ConfigMap from namepace")
			}

			// Deleting the configmap also triggers post-deploy actions
			if len(job.PostDeploy) > 0 {
				job.processPostDeployActions(logEntryBase, job.PostDeploy)
			}
			continue
		}

		logEntryBase.WithFields(logrus.Fields{
			"actualLabels":   cm.Metadata.Labels,
			"expectedLabels": job.Selector,
			"eventType":      eventType,
		}).Info("Found matching ConfigMap")

		if len(job.Verify) > 0 {
			logEntryBase.WithField("verifySteps", job.Verify).Debug("Verifying ConfigMap against validity")

			// TODO move this into separate function
			// Verify validity of ConfigMap files
			verifiedFiles, latestOutput, err := verifyCM(cm, job.Verify)
			if err != nil {
				logEntryBase.WithFields(logrus.Fields{
					"verifySteps":   job.Verify,
					"verifiedFiles": verifiedFiles,
					"latestOutput":  latestOutput,
					"error":         err,
				}).Warn("Failed to verify ConfigMap, there's something wrong with it, so we skip it..") //TODO document that we won't remove files that aren't valid anymore
				continue
			} else {
				logEntryBase.WithFields(logrus.Fields{
					"verifySteps":   job.Verify,
					"verifiedFiles": verifiedFiles,
				}).Debug("Successfully verified ConfigMap, ready to register")
			}
		}

		// ConfigMap has been verified, write files to filesystem
		registeredFiles, err := RegisterCM(cm, job.TargetDir, job.Flatten)
		logEntry := logEntryBase.WithFields(logrus.Fields{
			"eventType":       eventType,
			"registeredFiles": registeredFiles,
		})
		if err != nil {
			logEntry.WithError(err).Fatal("Failed to register ConfigMap")
		} else {
			logEntry.Info("Successfully registered ConfigMap")
		}

		// ConfigMap has ben registered, now run (optional) user-defined post deployment actions
		if len(job.PostDeploy) > 0 {
			job.processPostDeployActions(logEntryBase, job.PostDeploy)
		}
	}
}

func (job *Job) processPostDeployActions(logEntryBase *logrus.Entry, postDeployActions []PostDeployAction) {
	for _, postDeployAction := range postDeployActions {

		output, err := RunPostDeployActionCmd(postDeployAction.Cmd)

		logEntry := *logEntryBase.WithFields(logrus.Fields{
			"postDeployAction": postDeployAction,
			"output":           output,
		})
		if err != nil {
			logEntry.WithError(err).Error("Failed to executed postDeployAction command")
		} else {
			logEntry.Info("Successfully executed postDeployAction command")
		}
	}
}

// GetJob Translates the YAML main configuration file into a Job struct
func GetJob(log *logrus.Logger, configFile string) (Job, error) {
	yamlFile, err := ioutil.ReadFile(configFile)

	if yamlFile == nil {
		log.WithError(err).Fatalf("Error reading config file")
	}

	job := Job{}
	err = yaml.Unmarshal(yamlFile, &job)

	return job, err
}

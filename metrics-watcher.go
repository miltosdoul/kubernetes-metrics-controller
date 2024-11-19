package main

import (
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	"os"
	"time"
)

func main() {
	configPath := os.Getenv("KUBECONFIG")

	var (
		config *rest.Config
	)

	if configPath != "" {
		c, err := clientcmd.BuildConfigFromFlags("", configPath)
		if err != nil {
			panic(err)
		}
		config = c
	} else {
		panic("Environment variable KUBECONFIG not set")
	}

	clientSet := kubernetes.NewForConfigOrDie(config)
	metricsClientSet := metricsv.NewForConfigOrDie(config)
	sharedInformers := informers.NewSharedInformerFactory(clientSet, 2*time.Minute)

	limitController := NewLimitController(clientSet, metricsClientSet, sharedInformers.Core().V1().Pods())

	sharedInformers.Start(nil)
	limitController.Run(nil)
}

package client

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

type V1Beta1ClientWrapper struct {
	client         *versioned.Clientset
	PodMetricsList *v1beta1.PodMetricsList
}

func NewV1Beta1ClientWrapper(client *versioned.Clientset) *V1Beta1ClientWrapper {
	return &V1Beta1ClientWrapper{
		client:         client,
		PodMetricsList: doRetrievePodMetrics(client),
	}
}

func (c *V1Beta1ClientWrapper) List() *[]v1beta1.PodMetrics {
	return &c.PodMetricsList.Items
}

func (c *V1Beta1ClientWrapper) ListWithRefresh() *[]v1beta1.PodMetrics {
	c.UpdateList()
	return &c.PodMetricsList.Items
}

func (c *V1Beta1ClientWrapper) UpdateList() {
	c.PodMetricsList = doRetrievePodMetrics(c.client)
}

func doRetrievePodMetrics(client *versioned.Clientset) *v1beta1.PodMetricsList {
	l, err := client.MetricsV1beta1().PodMetricses("").List(context.TODO(), v1.ListOptions{
		FieldSelector: NoSystemFieldSelector,
	})

	if err != nil {
		// TODO: Introduce retries
		panic(fmt.Sprintf("Unable to get pods. Reason: %s. Exiting...\n", err))
	}

	return l
}

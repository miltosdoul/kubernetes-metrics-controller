package client

import (
	"context"
	"fmt"
	clientv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type V1ClientWrapper struct {
	client  *kubernetes.Clientset
	PodList *clientv1.PodList
}

func NewV1ClientWrapper(client *kubernetes.Clientset) *V1ClientWrapper {
	return &V1ClientWrapper{
		client:  client,
		PodList: doRetrievePods(client),
	}
}

func (c *V1ClientWrapper) List() *[]clientv1.Pod {
	return &c.PodList.Items
}

func (c *V1ClientWrapper) ListWithRefresh() *[]clientv1.Pod {
	c.UpdateList()
	return &c.PodList.Items
}

func (c *V1ClientWrapper) UpdateList() {
	c.PodList = doRetrievePods(c.client)
}

func doRetrievePods(client *kubernetes.Clientset) *clientv1.PodList {
	l, err := client.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{
		FieldSelector: NoSystemFieldSelector,
	})

	if err != nil {
		// TODO: Introduce retries
		panic(fmt.Sprintf("Unable to get pods. Reason: %s. Exiting...\n", err))
	}

	return l
}

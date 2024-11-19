package main

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
	"log"
	"metrics-watcher/client"
	"sync"
	"time"
)

type LimitController struct {
	podGetter       *client.V1ClientWrapper
	podLister       listersv1.PodLister
	podListerSynced cache.InformerSynced
	metricsGetter   *client.V1Beta1ClientWrapper
	// metrics API doesn't have lister

	queue workqueue.TypedRateLimitingInterface[string]
}

const (
	syncKey = "syncKey"
)

func NewLimitController(coreClient *kubernetes.Clientset, metricsClient *versioned.Clientset, podInformer informers.PodInformer) *LimitController {
	c := &LimitController{
		podGetter:       client.NewV1ClientWrapper(coreClient),
		podLister:       podInformer.Lister(),
		podListerSynced: podInformer.Informer().HasSynced,
		metricsGetter:   client.NewV1Beta1ClientWrapper(metricsClient),

		queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](workqueue.DefaultTypedControllerRateLimiter[string](), workqueue.TypedRateLimitingQueueConfig[string]{
			Name: "metrics queue",
		}),
	}

	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("Pod added")
				c.ScheduleSync()
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("Pod updated")
				c.ScheduleSync()
			},
			DeleteFunc: func(obj interface{}) {
				log.Print("Pod deleted")
				c.ScheduleSync()
			},
		},
	)

	return c
}

func (c *LimitController) Run(stopCh <-chan struct{}) {
	var wg sync.WaitGroup

	defer func() {
		// make sure the work queue is shut down which will trigger workers to end
		log.Print("shutting down queue")
		c.queue.ShutDown()

		// wait on the workers
		log.Print("shutting down workers")
		wg.Wait()

		log.Print("workers are all done")
	}()

	log.Print("waiting for cache sync")
	if !cache.WaitForCacheSync(stopCh, c.podListerSynced) {
		log.Print("timed out waiting for cache sync")
		return
	}
	log.Print("caches are synced")

	go func() {
		// runWorker will loop until "something bad" happens. wait.Until will
		// then rekick the worker after one second.
		wait.Until(c.runWorker, time.Second, stopCh)
		// tell the WaitGroup this worker is done
		wg.Done()
	}()

	// wait until we're told to stop
	log.Print("waiting for stop signal")
	<-stopCh
	log.Print("received stop signal")
}

func (c *LimitController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *LimitController) processNextWorkItem() bool {
	// pull the next work item from queue.  It should be a key we use to lookup
	// something in a cache
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// you always have to indicate to the queue that you've completed a piece of
	// work
	defer c.queue.Done(key)

	// do your work on the key.  This method will contains your "do stuff" logic
	err := c.doSync()
	if err == nil {
		// if you had no error, tell the queue to stop tracking history for your
		// key. This will reset things like failure counts for per-item rate
		// limiting
		c.queue.Forget(key)
		return true
	}

	// there was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring
	runtime.HandleError(fmt.Errorf("doSync failed with: %v", err))

	// since we failed, we should requeue the item to work on later.  This
	// method will add a backoff to avoid hotlooping on particular items
	// (they're probably still not going to work right away) and overall
	// controller protection (everything I've done is broken, this controller
	// needs to calm down or it can starve other useful work) cases.
	c.queue.AddRateLimited(key)

	return true
}

func (c *LimitController) ScheduleSync() {
	c.queue.Add(syncKey)
}

func (c *LimitController) doSync() error {
	log.Printf("Starting doSync")
	pods := c.podGetter.ListWithRefresh()
	podsWithLimits := c.getPodsWithLimits(pods)

	metrics := c.metricsGetter.ListWithRefresh()
	limitedPodsMetrics := c.getLimitedPodsMetrics(metrics, podsWithLimits)

	for i, m := range limitedPodsMetrics {
		fmt.Println("Pod ", i, ": ")
		fmt.Println("\tname: ", m.Name)
		fmt.Println("\tnamespace: ", m.Namespace)
		// TODO: for each container
		fmt.Println("\tCPU usage: ", m.Containers[0].Usage.Cpu())
		fmt.Println("\tCPU limit: ", podsWithLimits[m.Name].Spec.Containers[0].Resources.Limits.Cpu())
		fmt.Printf("\tCPU usage percentage: %.2f%%\n", calculateUsage(m.Containers[0].Usage.Cpu(), podsWithLimits[m.Name].Spec.Containers[0].Resources.Limits.Cpu()))
		fmt.Println("\tMemory usage: ", m.Containers[0].Usage.Memory())
		fmt.Println("\tMemory limit: ", podsWithLimits[m.Name].Spec.Containers[0].Resources.Limits.Memory())
		fmt.Printf("\tMemory usage percentage: %.2f%%\n", calculateUsage(m.Containers[0].Usage.Memory(), podsWithLimits[m.Name].Spec.Containers[0].Resources.Limits.Memory()))
	}

	log.Print("Finishing doSync")
	return nil
}

func calculateUsage(used *resource.Quantity, limit *resource.Quantity) float64 {
	return 100 * (used.AsApproximateFloat64() / limit.AsApproximateFloat64())
}

func (c *LimitController) getPodsWithLimits(podsList *[]v1.Pod) map[string]*v1.Pod {
	ret := make(map[string]*v1.Pod)

	for _, pod := range *podsList {
		for _, co := range pod.Spec.Containers {
			if len(co.Resources.Limits) != 0 {
				ret[pod.Name] = &pod
			}
		}
	}

	return ret
}

func (c *LimitController) getLimitedPodsMetrics(podMetricsList *[]v1beta1.PodMetrics, limitedPodsList map[string]*v1.Pod) (ret []v1beta1.PodMetrics) {
	for _, podMetric := range *podMetricsList {
		if limitedPodsList[podMetric.Name] != nil {
			ret = append(ret, podMetric)
		}
	}

	return ret
}

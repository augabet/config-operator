package controller

import (
	"fmt"
	"log"
	"time"

	"github.com/bitnami-labs/kubewatch/pkg/handlers"
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// ConfigMapController watches the kubernetes api for changes to configmap and
// creates a RoleBinding for that particular namespace.
type ConfigMapController struct {
	informer     cache.SharedIndexInformer
	kclient      *kubernetes.Clientset
	queue        workqueue.RateLimitingInterface
	eventHandler handlers.Handler
}

// Run starts the process for listening for configmap changes and acting upon those changes.
func (c *ConfigMapController) Run(stopCh <-chan struct{}) {
	// don't let panics crash the process
	defer utilruntime.HandleCrash()
	// make sure the work queue is shutdown which will trigger workers to end
	defer c.queue.ShutDown()

	go c.informer.Run(stopCh)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	// runWorker will loop until "something bad" happens.  The .Until will
	// then rekick the worker after one second
	wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *ConfigMapController) runWorker() {
	// processNextWorkItem will automatically wait until there's work available
	for c.processNextItem() {
		// continue looping
	}
}

func (c *ConfigMapController) HasSynced() bool {
	return c.informer.HasSynced()
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
const maxRetries = 5

func (c *ConfigMapController) processNextItem() bool {
	// pull the next work item from queue.  It should be a key we use to lookup
	// something in a cache
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	// you always have to indicate to the queue that you've completed a piece of
	// work
	defer c.queue.Done(key)

	// do your work on the key.
	err := c.processItem(key.(string))

	if err == nil {
		// No error, tell the queue to stop tracking history
		c.queue.Forget(key)
	} else if c.queue.NumRequeues(key) < maxRetries {
		// requeue the item to work on later
		c.queue.AddRateLimited(key)
	} else {
		// err != nil and too many retries
		c.queue.Forget(key)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *ConfigMapController) processItem(key string) error {

	obj, _, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("Error fetching object with key %s from store: %v", key, err)
	}
	c.eventHandler.ObjectUpdated("", obj)
	//c.eventHandler.reconcile(_, obj)
	return nil
}

func (c *ConfigMapController) ObjectUpdated(_, obj interface{}) {

	log.Println(fmt.Sprintf("TOTOTOTOTOTOTOTOTOTO"))
	// c.createRoleBinding(obj)
}

// NewConfigMapController creates a new ConfigMapController
func NewConfigMapController(kclient *kubernetes.Clientset) *ConfigMapController {
	configMapWatcher := &ConfigMapController{}
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Create informer for watching configMap
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kclient.Core().ConfigMaps(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kclient.Core().ConfigMaps(metav1.NamespaceAll).Watch(options)
			},
		},
		&v1.ConfigMap{},
		3*time.Minute,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		//		UpdateFunc: configMapWatcher.reconcile,
	})

	configMapWatcher.kclient = kclient
	configMapWatcher.informer = informer

	return configMapWatcher
}

// func (c *ConfigMapController) createRoleBinding(obj interface{}) {
// 	configmapObj := obj.(*v1.ConfigMap)
// 	namespaceName := configmapObj.Namespace

// 	roleBinding := &v1beta1.RoleBinding{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind:       "RoleBinding",
// 			APIVersion: "rbac.authorization.k8s.io/v1beta1",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      fmt.Sprintf("ad-kubernetes-%s", namespaceName),
// 			Namespace: namespaceName,
// 		},
// 		Subjects: []v1beta1.Subject{
// 			v1beta1.Subject{
// 				Kind: "Group",
// 				Name: fmt.Sprintf("ad-kubernetes-%s", namespaceName),
// 			},
// 		},
// 		RoleRef: v1beta1.RoleRef{
// 			APIGroup: "rbac.authorization.k8s.io",
// 			Kind:     "ClusterRole",
// 			Name:     "edit",
// 		},
// 	}

// 	_, err := c.kclient.Rbac().RoleBindings(namespaceName).Create(roleBinding)

// 	if err != nil {
// 		log.Println(fmt.Sprintf("Failed to create Role Binding: %s", err.Error()))
// 	} else {
// 		log.Println(fmt.Sprintf("Created AD RoleBinding for Namespace: %s", roleBinding.Name))
// 	}
// }

// func (c *ConfigMapController) reconcile(_, obj interface{}) {
// 	cm := obj.(*v1.ConfigMap)
// 	ns := cm.Namespace

// 	deployList, err := c.kclient.Apps().Deployments(ns).List(metav1.ListOptions{})
// 	if err != nil {
// 		panic(err)
// 	}

// 	var referCMList []v1beta2.Deployment
// 	for _, deploy := range deployList.Items {
// 		if isRefered(cm.Name, deploy) {
// 			referCMList = append(referCMList, deploy)
// 		}
// 	}

// 	for _, deploy := range referCMList {
// 		err = c.triggerRollingUpdate(cm.UID, deploy)
// 	}
// }

func isRefered(cmName string, deploy v1beta2.Deployment) bool {
	return true
}

func (c *ConfigMapController) triggerRollingUpdate(cmUID types.UID, deploy v1beta2.Deployment) error {
	return nil
}

package controller

import (
	"sync"
	"time"

	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ConfigMapController watches the kubernetes api for changes to configmap and
// creates a RoleBinding for that particular namespace.
type ConfigMapController struct {
	configMapInformer cache.SharedIndexInformer
	kclient           *kubernetes.Clientset
}

// Run starts the process for listening for configmap changes and acting upon those changes.
func (c *ConfigMapController) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	// When this function completes, mark the go function as done
	defer wg.Done()

	// Increment wait group as we're about to execute a go function
	wg.Add(1)

	// Execute go function
	go c.configMapInformer.Run(stopCh)

	// Wait till we receive a stop signal
	<-stopCh
}

// NewConfigMapController creates a new ConfigMapController
func NewConfigMapController(kclient *kubernetes.Clientset) *ConfigMapController {
	configMapWatcher := &ConfigMapController{}

	// Create informer for watching configMap
	configMapInformer := cache.NewSharedIndexInformer(
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

	configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: configMapWatcher.reconcile,
	})

	configMapWatcher.kclient = kclient
	configMapWatcher.configMapInformer = configMapInformer

	return configMapWatcher
}

func (c *ConfigMapController) reconcile(_, obj interface{}) {
	cm := obj.(*v1.ConfigMap)
	ns := cm.Namespace

	deployList, err := c.kclient.Apps().Deployments(ns).List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	var referCMList []v1beta2.Deployment
	for _, deploy := range deployList.Items {
		if isRefered(cm.Name, deploy) {
			referCMList = append(referCMList, deploy)
		}
	}

	for _, deploy := range referCMList {
		err = c.triggerRollingUpdate(cm.UID, deploy)
	}
}

func isRefered(cmName string, deploy v1beta2.Deployment) bool {
	return true
}

func (c *ConfigMapController) triggerRollingUpdate(cmUID types.UID, deploy v1beta2.Deployment) error {
	return nil
}

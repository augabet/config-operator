package controller

import (
	"fmt"
	"log"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		AddFunc: configMapWatcher.createRoleBinding,
	})

	configMapWatcher.kclient = kclient
	configMapWatcher.configMapInformer = configMapInformer

	return configMapWatcher
}

func (c *ConfigMapController) createRoleBinding(obj interface{}) {
	configmapObj := obj.(*v1.ConfigMap)
	namespaceName := configmapObj.Namespace

	roleBinding := &v1beta1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ad-kubernetes-%s", namespaceName),
			Namespace: namespaceName,
		},
		Subjects: []v1beta1.Subject{
			v1beta1.Subject{
				Kind: "Group",
				Name: fmt.Sprintf("ad-kubernetes-%s", namespaceName),
			},
		},
		RoleRef: v1beta1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "edit",
		},
	}

	_, err := c.kclient.Rbac().RoleBindings(namespaceName).Create(roleBinding)

	if err != nil {
		log.Println(fmt.Sprintf("Failed to create Role Binding: %s", err.Error()))
	} else {
		log.Println(fmt.Sprintf("Created AD RoleBinding for Namespace: %s", roleBinding.Name))
	}
}

package controller

import (
	"context"
	"fmt"
	ash_dev "github.com/AshrafulHaqueToni/crd-controller/pkg/apis/ash.dev"
	controllerv1 "github.com/AshrafulHaqueToni/crd-controller/pkg/apis/ash.dev/v1alpha1"
	clientset "github.com/AshrafulHaqueToni/crd-controller/pkg/client/clientset/versioned"
	informer "github.com/AshrafulHaqueToni/crd-controller/pkg/client/informers/externalversions/ash.dev/v1alpha1"
	lister "github.com/AshrafulHaqueToni/crd-controller/pkg/client/listers/ash.dev/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log"
	"strings"
	"time"
)

type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	sampleclientset   clientset.Interface
	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	ashLister         lister.AshLister
	ashSynced         cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workQueue workqueue.RateLimitingInterface
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	deploymentInformer appsinformer.DeploymentInformer,
	ashInformer informer.AshInformer) *Controller {
	ctrl := &Controller{
		kubeclientset:     kubeclientset,
		sampleclientset:   sampleclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		ashLister:         ashInformer.Lister(),
		ashSynced:         ashInformer.Informer().HasSynced,
		workQueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Ashes"),
	}

	log.Println("Setting up eventhandler")

	// Set up an event handler for when Ash resources change

	ashInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ctrl.enqueueAsh,
		UpdateFunc: func(oldObj, newObj interface{}) {
			ctrl.enqueueAsh(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			ctrl.enqueueAsh(obj)
		},
	})

	return ctrl

}

// enqueueAsh takes an Ash resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Ash.

func (c *Controller) enqueueAsh(obj interface{}) {
	log.Println("Enqueuing Ash")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workQueue.AddRateLimited(key)
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShuttingDown()

	// start the informer factories to begin populating the informer caches
	log.Println("Start Ash Controller ")
	// Wait for the caches to be synced before starting workers
	log.Println("waiting for informer cache to sync")

	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.ashSynced); !ok {
		return fmt.Errorf("fail to wait for cache to sync")
	}

	log.Println("Staring workers")

	// Launch two workers to process Ash resources

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Println("Worker started")

	<-stopCh
	log.Println("shutting down workers")

	return nil

}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the workqueue.
func (c *Controller) runWorker() {
	for c.ProcessNextItem() {
	}
}

func (c *Controller) ProcessNextItem() bool {
	obj, shutdown := c.workQueue.Get()
	if shutdown {
		return false
	}
	// We wrap this block in a func, so we can defer c.workqueue Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off period.
		defer c.workQueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passoing it the namespace/name string of the
		// Kluster resource to be synced.

		if err := c.snycHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		// Finally, if no error occurs we Forget this item, so it does not
		// get queued again until another change happens.
		c.workQueue.Forget(obj)
		log.Printf("successfully synced '%s'\n", key)

		return nil

	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true

}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Pritam resource
// with the current status of the resource.
// implement the business logic here.

func (c *Controller) snycHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	// Get Ash resource with this namespcae/name
	ash, err := c.ashLister.Ashes(namespace).Get(name)
	if err != nil {
		// The kluster resource may no longer exist, in which case we stop processing.
		if errors.IsNotFound(err) {
			// We choose to absorb the error here as the worker would requeue the
			// resource otherwise. Instead, the next time the resource is updated
			// the resource will be queued again.
			utilruntime.HandleError(fmt.Errorf("ash '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	log.Println("we are here ", ash.Spec.Name)

	if err = c.DeploymentHandler(ash); err != nil {
		utilruntime.HandleError(fmt.Errorf("error while handling deployment: %s", err.Error()))
	}

	if err = c.ServiceHandler(ash); err != nil {
		utilruntime.HandleError(fmt.Errorf("error while handling service: %s", err.Error()))
	}

	return nil
}

func (c *Controller) DeploymentHandler(ash *controllerv1.Ash) error {
	deploymentName := ash.Spec.Name

	if ash.Spec.Name == "" {
		deploymentName = strings.Join(buildSlice(ash.Name, ash_dev.Deployment), "-")
	}
	namespace := ash.Namespace
	name := ash.Name

	deployment, err := c.deploymentsLister.Deployments(namespace).Get(deploymentName)
	// if the resource doesn't exist we'll create it
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclientset.AppsV1().Deployments(ash.Namespace).Create(context.TODO(), c.newDeployment(ash), metav1.CreateOptions{})
		if err != nil {
			return err
		}
		log.Println("deployment name here for ash dot dev", deploymentName)
	}
	// If an error occurs during Get/Create, we'll requeue the item, so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If this number of the replicas on the kluster resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	if ash.Spec.Replicas != nil && *ash.Spec.Replicas != *deployment.Spec.Replicas {
		log.Printf("Ash %s replicas: %d, deployment replicas: %d\n", name, *ash.Spec.Replicas, *deployment.Spec.Replicas)
		*deployment.Spec.Replicas = *ash.Spec.Replicas
		deployment, err = c.kubeclientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})

		// If an error occurs during Update, we'll requeue the item, so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		if err != nil {
			return err
		}
	}
	// Finally, we update the status block of the Pritam resource to reflect the
	// current state of the world
	log.Println("we are updating status here")
	err = c.updateAshStatus(ash, deployment)
	if err != nil {
		return err
	}
	return nil

}

func (c *Controller) ServiceHandler(ash *controllerv1.Ash) error {
	serviceName := ash.Spec.Name

	if ash.Spec.Name == "" {
		serviceName = strings.Join(buildSlice(ash.Name, ash_dev.Service), "-")
	}

	service, err := c.kubeclientset.CoreV1().Services(ash.Namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	// if the resource doesn't exist we'll create it
	if errors.IsNotFound(err) {
		service, err = c.kubeclientset.CoreV1().Services(ash.Namespace).Create(context.TODO(), c.newService(ash), metav1.CreateOptions{})
		if err != nil {
			log.Println(err)
			return err
		}
		log.Printf("\n service %s created .....\n", service.Name)
	} else if err != nil {
		log.Println(err)
		return err
	}
	_, err = c.kubeclientset.CoreV1().Services(ash.Namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (c *Controller) updateAshStatus(ash *controllerv1.Ash, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	ashCopy := ash.DeepCopy()

	ashCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas

	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Foo resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.

	_, err := c.sampleclientset.AshV1alpha1().Ashes(ash.Namespace).Update(context.TODO(), ashCopy, metav1.UpdateOptions{})

	return err

}

func buildSlice(arr ...string) []string {
	var ss []string

	for _, v := range arr {
		ss = append(ss, v)
	}
	return ss
}

func (c *Controller) newDeployment(ash *controllerv1.Ash) *appsv1.Deployment {
	log.Println("we have name in newdeployment", ash.Spec.Name)
	deploymentName := ash.Spec.Name
	if ash.Spec.Name == "" {
		deploymentName = strings.Join(buildSlice(ash.Name, ash_dev.Deployment), "-")
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: ash.Namespace,
			OwnerReferences: []metav1.OwnerReference{

				*metav1.NewControllerRef(ash, controllerv1.SchemeGroupVersion.WithKind(ash_dev.ResourceDefinition)),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ash.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					ash_dev.App:     ash_dev.Myapp,
					ash_dev.AshName: ash.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						ash_dev.App:     ash_dev.Myapp,
						ash_dev.AshName: ash.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-app",
							Image: ash.Spec.Container.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: ash.Spec.Container.Port,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *Controller) newService(ash *controllerv1.Ash) *corev1.Service {
	serviceName := ash.Spec.Name
	if ash.Spec.Name == "" {
		serviceName = strings.Join(buildSlice(ash.Name, ash_dev.Service), "-")
	}
	labels := map[string]string{
		ash_dev.App:     ash_dev.Myapp,
		ash_dev.AshName: ash.Name,
	}
	return &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ash, controllerv1.SchemeGroupVersion.WithKind(ash_dev.ResourceDefinition)),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       ash.Spec.Container.Port,
					TargetPort: intstr.FromInt(int(ash.Spec.Container.Port)),
				},
			},
		},
	}
}

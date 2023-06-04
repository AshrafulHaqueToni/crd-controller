package main

import (
	"flag"
	"github.com/AshrafulHaqueToni/crd-controller/controller"
	clientset "github.com/AshrafulHaqueToni/crd-controller/pkg/client/clientset/versioned"
	informers "github.com/AshrafulHaqueToni/crd-controller/pkg/client/informers/externalversions"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
	"time"
)

func main() {
	log.Println("configure kubeconfig")
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to kubeconfig file")
	}
	log.Println(kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	sampleClient, err := clientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	// Initialise the informer resource and here we will be using sharedinformer factory instead of simple informers
	// because in case if we need to query / watch multiple Group versions, and it’s a good practise as well
	// NewSharedInformerFactory will create a new ShareInformerFactory for "all namespaces"
	// 30*time.Second is the re-sync period to update the in-memory cache of informer //

	kubeInformationFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	sampleInformationFactory := informers.NewSharedInformerFactory(sampleClient, time.Second*30)

	log.Println(kubeInformationFactory, sampleInformationFactory)

	// From this informerfactory we can create specific informers for every group version resource
	// that are default available in k8s environment such as Pods, deployment, etc
	// podInformer := kubeInformationFactory.Core().V1().Pods()
	ctrl := controller.NewController(kubeClient, sampleClient,
		kubeInformationFactory.Apps().V1().Deployments(),
		sampleInformationFactory.Ash().V1alpha1().Ashes())
	stopCh := make(chan struct{})
	kubeInformationFactory.Start(stopCh)
	sampleInformationFactory.Start(stopCh)

	if err = ctrl.Run(2, stopCh); err != nil {
		log.Println("Error running controller")
	}
}

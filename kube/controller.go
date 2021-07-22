package kube

import (
	"context"
	"fmt"
	"net"
	"time"

	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeInformers "k8s.io/client-go/informers"
	informerAppsV1 "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appsListers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	"github.com/xwen-winnie/crd_demo/kube/apis/qbox/v1alpha1"
	clientSet "github.com/xwen-winnie/crd_demo/kube/client/clientset/versioned"
	customScheme "github.com/xwen-winnie/crd_demo/kube/client/clientset/versioned/scheme"
	externalInformer "github.com/xwen-winnie/crd_demo/kube/client/informers/externalversions"
	qboxInformer "github.com/xwen-winnie/crd_demo/kube/client/informers/externalversions/qbox/v1alpha1"
	qboxListers "github.com/xwen-winnie/crd_demo/kube/client/listers/qbox/v1alpha1"
	"github.com/xwen-winnie/crd_demo/utils"
)

const (
	defaultResync       = time.Second * 30
	controllerAgentName = "wait-deployment-controller"
	WaitdeploymentKind  = "Waitdeployment"

	SuccessSynced         = "Synced"
	ErrResourceExists     = "ErrResourceExists"
	MessageResourceSynced = "Waitdeployment synced successfully"
	MessageResourceExists = "Resource %q already exists and is not managed by Waitdeployment"
)

type Client struct {
	ctx context.Context

	kubeClient         kubernetes.Interface
	customClient       clientSet.Interface
	kubeFactory        kubeInformers.SharedInformerFactory
	customFactory      externalInformer.SharedInformerFactory
	deploymentInformer informerAppsV1.DeploymentInformer
	customInformer     qboxInformer.WaitdeploymentInformer

	deploymentsLister     appsListers.DeploymentLister
	deploymentsSynced     cache.InformerSynced
	waitDeploymentsLister qboxListers.WaitdeploymentLister
	waitDeploymentsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewClient(ctx context.Context, path string) (*Client, error) {
	var cfg Config
	if err := utils.LoadYAML(path, &cfg); err != nil {
		return nil, err
	}

	kubeConfig, err := func() (*rest.Config, error) {
		if !cfg.Kube.OutCluster {
			return rest.InClusterConfig()
		}
		return clientcmd.BuildConfigFromFlags(
			"", cfg.Kube.ConfigPath)
	}()
	if err != nil {
		return nil, err
	}

	// client
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	customClient, err := clientSet.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	// factory
	kubeFactory := kubeInformers.NewSharedInformerFactory(kubeClient, defaultResync)
	customFactory := externalInformer.NewSharedInformerFactory(customClient, defaultResync)

	// informer
	deploymentInformer := kubeFactory.Apps().V1().Deployments()
	customInformer := customFactory.Qbox().V1alpha1().Waitdeployments()

	// Create event broadcaster
	// Add my-controller types to the default Kubernetes Scheme so Events can be
	// logged for my-controller types.
	utilRuntime.Must(customScheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedCoreV1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, coreV1.EventSource{Component: controllerAgentName})

	client := &Client{
		ctx: ctx,

		kubeClient:         kubeClient,
		customClient:       customClient,
		kubeFactory:        kubeFactory,
		customFactory:      customFactory,
		deploymentInformer: deploymentInformer,
		customInformer:     customInformer,

		deploymentsLister:     deploymentInformer.Lister(),
		deploymentsSynced:     deploymentInformer.Informer().HasSynced,
		waitDeploymentsLister: customInformer.Lister(),
		waitDeploymentsSynced: customInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), WaitdeploymentKind),
		recorder:              recorder,
	}

	client.registerEventHandler()
	return client, nil
}

func (c *Client) registerEventHandler() {
	// Set up an event handler for when waitDeployment resources change
	c.customInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueWaitdeployment,
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.enqueueWaitdeployment(newObj)
		},
	})

	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a waitDeployment resource will enqueue that waitDeployment resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	c.deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(oldObj, newObj interface{}) {
			newDeployment := newObj.(*appsV1.Deployment)
			oldDeployment := oldObj.(*appsV1.Deployment)
			if newDeployment.ResourceVersion == oldDeployment.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			c.handleObject(newObj)
		},
		DeleteFunc: c.handleObject,
	})
}

func (c *Client) Start(stopCh <-chan struct{}) {
	c.kubeFactory.Start(stopCh)
	c.customFactory.Start(stopCh)
}

// 启动controller
// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Client) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilRuntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Waitdeployment controller")

	// 在worker运行之前，必须要等待状态的同步完成
	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.waitDeploymentsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// 启动多个 worker 协程并发从 queue 中获取需要处理的 item
	// runWorker 是包含真正的业务逻辑的函数
	// Launch n workers to process Waitdeployment resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Client) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem 从 workqueue 中获取一个任务并最终调用 syncHandler 执行她
func (c *Client) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	// 这里写成函数形式是为了方便里面能直接调用 defer
	err := func(obj interface{}) error {
		// 通过调用 Done 方法可以通知 workqueue 完成了这个任务
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// 通过调用 Forget 方法可以避免任务被再次入队，比如调用一个任务出错后，为了避免
			// 它再次放入队列底部并在 back-off 后再次尝试，可以调用这个方法
			c.workqueue.Forget(obj)
			utilRuntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilRuntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Waitdeployment
// resource with the current status of the resource.
func (c *Client) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	ns, n, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	waitDeployment, err := c.waitDeploymentsLister.Waitdeployments(ns).Get(n)
	if err != nil {
		// The Waitdeployment resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf("Waitdeployment '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	deployment, err := c.checkAndStartDeployment(waitDeployment)
	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this waitDeployment resource,
	// we should log a warning to the event recorder and return error msg.
	if !metav1.IsControlledBy(deployment, waitDeployment) {
		msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
		c.recorder.Event(waitDeployment, coreV1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// update deployment
	deployment, err = c.kubeClient.AppsV1().Deployments(waitDeployment.Namespace).Update(c.ctx, newDeployment(waitDeployment), metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// update waitDeployment
	_, err = c.customClient.QboxV1alpha1().Waitdeployments(waitDeployment.Namespace).Update(c.ctx, waitDeployment.DeepCopy(), metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	c.recorder.Event(waitDeployment, coreV1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Client) checkAndStartDeployment(waitDeployment *v1alpha1.Waitdeployment) (*appsV1.Deployment, error) {
	err := doTCPProbe(waitDeployment.WaitProbe.Address, waitDeployment.WaitProbe.Timeout)
	if err != nil {
		return nil, err
	}
	deployment, err := c.deploymentsLister.Deployments(waitDeployment.Namespace).Get(waitDeployment.Name)
	if errors.IsNotFound(err) {
		klog.Infof("Waitdeployment not exist, create a new deployment %s in namespace %s", waitDeployment.Name, waitDeployment.Namespace)
		deployment, err = c.kubeClient.AppsV1().Deployments(waitDeployment.Namespace).Create(c.ctx, newDeployment(waitDeployment), metav1.CreateOptions{})
	}
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

// tcp 连接检测代码
func doTCPProbe(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	err = conn.Close()
	if err != nil {
		klog.Errorf("Unexpected error closing TCP probe socket: %v (%#v)", err, err)
	}
	return nil
}

func newDeployment(waitDeployment *v1alpha1.Waitdeployment) *appsV1.Deployment {
	res := &appsV1.Deployment{
		TypeMeta:   waitDeployment.TypeMeta,
		ObjectMeta: *waitDeployment.ObjectMeta.DeepCopy(),
		Spec:       waitDeployment.Spec,
		Status:     waitDeployment.Status,
	}
	res.ResourceVersion = ""
	res.UID = ""
	res.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(waitDeployment, v1alpha1.SchemeGroupVersion.WithKind(WaitdeploymentKind)),
	}
	res.Labels["controller"] = waitDeployment.Name
	res.Spec.Selector.MatchLabels["controller"] = waitDeployment.Name
	res.Spec.Template.Labels["controller"] = waitDeployment.Name
	return res
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Waitdeployment resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Waitdeployment resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
//
// 将任何实现 metav1.Object 的资源并尝试找到“拥有”它的 Waitdeployment 资源。
// 它通过查看对象 metadata.ownerReferences 字段以获取适当的 OwnerReference 来完成此操作。
// 然后将要处理的 Waitdeployment 资源加入队列。 如果对象没有合适的 OwnerReference，它将被简单地跳过。
func (c *Client) handleObject(obj interface{}) {
	object, ok := obj.(metav1.Object)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilRuntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilRuntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Foo, we should not do anything more with it.
		if ownerRef.Kind != WaitdeploymentKind {
			return
		}
		waitDeployment, err := c.waitDeploymentsLister.Waitdeployments(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}
		c.enqueueWaitdeployment(waitDeployment)
		return
	}
}

// enqueueWaitdeployment takes a enqueueWaitdeployment resource and converts
// it into a namespace/name string which is then put onto the work queue.
// This method should *not* be passed resources of any type other than enqueueWaitdeployment.
//
// 它需要一个 enqueueWaitdeployment 资源并将其转换为命名空间/名称字符串，然后将其放入工作队列。
// 此方法不应传递除 enqueueWaitdeployment 之外的任何类型的资源。
func (c *Client) enqueueWaitdeployment(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilRuntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

package pod

import (
	"context"
	"os"

	"github.com/ls-2018/k8s-watch/pkg/cfg"
	corev1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	cache2 "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func Add(mgr manager.Manager) error {
	r, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("pod-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}
	err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Pod{}, &handler.TypedEnqueueRequestForObject[*corev1.Pod]{}))
	if err != nil {
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	cacher := mgr.GetCache()
	podInformer, err := cacher.GetInformerForKind(context.TODO(), corev1.SchemeGroupVersion.WithKind("Pod"))
	if err != nil {
		return nil, err
	}

	podLister := corelisters.NewPodLister(podInformer.(cache.SharedIndexInformer).GetIndexer())

	dsc := &ReconcileDaemonSet{
		podLister:   podLister,
		podInformer: podInformer,
	}
	return dsc, err
}

type ReconcileDaemonSet struct {
	eventRecorder record.EventRecorder
	podLister     corelisters.PodLister
	podInformer   cache2.Informer
}

var _ reconcile.Reconciler = &ReconcileDaemonSet{}

func (r *ReconcileDaemonSet) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cache.WaitForCacheSync(ctx.Done(), r.podInformer.HasSynced)
	pod, err := r.podLister.Pods(request.Namespace).Get(request.Name)
	if err != nil {
		return reconcile.Result{}, nil
	}
	name, _ := os.Hostname()
	if request.Name == name && pod.Annotations != nil && pod.Annotations["template"] != "" {
		klog.Infoln("set template=", pod.Annotations["template"])
		cfg.Template.Store(pod.Annotations["template"])
	}
	return reconcile.Result{}, nil
}

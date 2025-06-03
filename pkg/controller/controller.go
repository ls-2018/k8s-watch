package controller

import (
	"errors"

	"github.com/ls-2018/k8s-watch/pkg/controller/pod"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var controllerAddFuncs []func(manager.Manager) error

func init() {
	controllerAddFuncs = append(controllerAddFuncs, pod.Add)
}

func SetupWithManager(m manager.Manager) error {
	for _, f := range controllerAddFuncs {
		if err := f(m); err != nil {
			var kindMatchErr *meta.NoKindMatchError
			if errors.As(err, &kindMatchErr) {
				klog.InfoS("CRD is not installed, its controller will perform noops!", "CRD", kindMatchErr.GroupKind)
				continue
			}
			return err
		}
	}
	return nil
}

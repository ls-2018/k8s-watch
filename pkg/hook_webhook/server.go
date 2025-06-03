package hook_webhook

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type GateFunc func() (enabled bool)

var (
	// HandlerMap contains all admission hook_webhook handlers.
	HandlerMap   = map[string]admission.Handler{}
	handlerGates = map[string]GateFunc{}
)

func addHandlers(m map[string]admission.Handler) {
	addHandlersWithGate(m, nil)
}

func addHandlersWithGate(m map[string]admission.Handler, fn GateFunc) {
	for path, handler := range m {
		if len(path) == 0 {
			klog.Warningf("Skip handler with empty path.")
			continue
		}
		if path[0] != '/' {
			path = "/" + path
		}
		_, found := HandlerMap[path]
		if found {
			klog.V(1).Infof("conflicting hook_webhook builder path %v in handler map", path)
		}
		HandlerMap[path] = handler
		if fn != nil {
			handlerGates[path] = fn
		}
	}
}

func filterActiveHandlers() {
	disablePaths := sets.NewString()
	for path := range HandlerMap {
		if fn, ok := handlerGates[path]; ok {
			if !fn() {
				disablePaths.Insert(path)
			}
		}
	}
	for _, path := range disablePaths.List() {
		delete(HandlerMap, path)
	}
}

func SetupWithManager(mgr manager.Manager) {
	server := mgr.GetWebhookServer()

	filterActiveHandlers()
	for path, handler := range HandlerMap {
		server.Register(path, &webhook.Admission{Handler: handler})
		klog.V(3).Infof("Registered hook_webhook handler %s", path)
	}

}

package mutating

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	HandlerMap = map[string]admission.Handler{
		"watch": &DynamicHook{},
	}
)

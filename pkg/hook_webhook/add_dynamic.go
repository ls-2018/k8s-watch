package hook_webhook

import (
	"github.com/ls-2018/k8s-watch/pkg/hook_webhook/dynamic/mutating"
)

func init() {
	addHandlersWithGate(mutating.HandlerMap, func() (enabled bool) {
		return true
	})
}

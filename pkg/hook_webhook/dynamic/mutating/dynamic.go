package mutating

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/ls-2018/k8s-watch/pkg/cfg"
	"github.com/tidwall/gjson"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type DynamicHook struct {
}

func (pch *DynamicHook) Handle(ctx context.Context, ar admission.Request) (reviewResponse admission.Response) {
	reviewResponse.Allowed = true
	oldStr := string(ar.OldObject.Raw)
	newStr := string(ar.Object.Raw)

	if cfg.Template.Load() == nil {
		return admission.PatchResponseFromRaw([]byte(""), []byte(""))
	}
	template := cfg.Template.Load().(string)
	if template != "" {
		oldTmp := gjson.Get(oldStr, template)
		newTmp := gjson.Get(newStr, template)

		klog.Infoln(fmt.Sprintf(
			"template:%s RequestKind:%s RequestResource:/%s RequestSubResource:/%s Obj:/%s/%s operator:%s username:%s userid:%s %s -> %s",
			base64.StdEncoding.EncodeToString([]byte(template)),
			ar.RequestKind.String(),
			ar.RequestResource.String(),
			ar.RequestSubResource,
			ar.Namespace, ar.Name, ar.Operation,
			ar.UserInfo.Username, ar.UserInfo.UID,
			oldTmp.String(), newTmp.String(),
		))
	}
	return admission.PatchResponseFromRaw([]byte(""), []byte(""))
}

/*
Copyright 2020 The Kruise Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"os"
	"strconv"
	"time"

	"k8s.io/klog/v2"
)

var (
	renewBefore time.Duration
)

func GetRenewBeforeTime() time.Duration {
	if renewBefore != 0 {
		return renewBefore
	}
	renewBefore = 6 * 30 * 24 * time.Hour
	if s := os.Getenv("CERTS_RENEW_BEFORE"); len(s) > 0 {
		t, err := strconv.Atoi(s[0 : len(s)-1])
		if err != nil {
			klog.ErrorS(err, "failed to parse time", "time", s[0:len(s)-1])
			return renewBefore
		}
		suffix := s[len(s)-1]
		if suffix == 'd' {
			renewBefore = time.Duration(t) * 7 * time.Hour
		} else if suffix == 'm' {
			renewBefore = time.Duration(t) * 30 * time.Hour
		} else if suffix == 'y' {
			renewBefore = time.Duration(t) * 365 * time.Hour
		} else {
			klog.InfoS("unknown date suffix", "suffix", suffix)
		}
	}
	if renewBefore <= 0 {
		klog.Error("renewBefore time can not be less or equal than 0")
		renewBefore = 6 * 30 * 24 * time.Hour
	}
	return renewBefore
}

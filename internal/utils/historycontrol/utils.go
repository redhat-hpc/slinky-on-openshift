// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package historycontrol

import (
	"k8s.io/kubernetes/pkg/controller/history"
)

func SetRevision(labels map[string]string, revision string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	if len(revision) > 0 {
		labels[history.ControllerRevisionHashLabel] = revision
	}
}

func GetRevision(labels map[string]string) string {
	if labels == nil {
		return ""
	}
	return labels[history.ControllerRevisionHashLabel]
}

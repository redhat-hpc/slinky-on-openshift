// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"cmp"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// Clamp returns a value such that it remain in range [min, max].
// NOTE: the clamped range will be determined from a, b inputs.
func Clamp[T cmp.Ordered](val, a, b T) T {
	lower := min(a, b)
	upper := max(a, b)
	return min(max(val, lower), upper)
}

// GetScaledValueFromIntOrPercent returns a scaled value given an int or percent,
// otherwise returns the default value.
func GetScaledValueFromIntOrPercent(intOrPercent *intstr.IntOrString, total int, roundUp bool, defaultValue int) int {
	val, err := intstr.GetScaledValueFromIntOrPercent(intOrPercent, total, roundUp)
	if err != nil {
		val = defaultValue
	}
	return val
}

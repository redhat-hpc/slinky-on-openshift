// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package durationstore

import (
	"sync"
	"time"
)

// DurationStore{} stores arbitrary time.Duration. It allows for safe pushes and
// pops based on any arbitrary keys.
type DurationStore struct {
	// Underlying data store
	sync.Map

	// Function to resolve which duration to keep when repeated push to same key
	eval func(oldDur, newDur time.Duration) bool
}

// NewDurationStore() returns a duration store which will evaluate which value to
// keep, when multiple are pushed to the same key, based on the eval() function.
func NewDurationStore(eval func(oldDur, newDur time.Duration) bool) *DurationStore {
	return &DurationStore{eval: eval}
}

func Greater(oldDur, newDur time.Duration) bool {
	return newDur > oldDur
}

func Less(oldDur, newDur time.Duration) bool {
	return newDur < oldDur
}

// Push() will store a duration for the key. If multiple values are pushed onto
// the same key before the key is popped, then the one that returns true from
// DurationStore.eval() is kept.
func (dm *DurationStore) Push(key string, newDur time.Duration) {
	newD := &duration{dur: newDur}
	val, loaded := dm.LoadOrStore(key, newD)
	if !loaded {
		return
	}
	d, ok := val.(*duration)
	if !ok {
		// edge case: corrupt stored duration
		// recover: delete key and store new duration
		dm.Delete(key)
		dm.Store(key, newD)
		return
	}
	d.Update(newDur, dm.eval)
}

// Pop() will return the duration stored by the key and delete the store. If no
// duration was stored for that key, then 0 will be returned.
func (dm *DurationStore) Pop(key string) time.Duration {
	val, ok := dm.LoadAndDelete(key)
	if !ok {
		// Nothing was stored, return 0
		return 0
	}
	d, ok := val.(*duration)
	if !ok {
		// edge case: corrupt stored duration
		// recover: assume nothing was stored, return 0
		return 0
	}
	return d.Get()
}

// Peek() will return the duration stored by the key and *not* delete the store.
// If no duration was stored for that key, then 0 will be returned.
func (dm *DurationStore) Peek(key string) time.Duration {
	val, ok := dm.Load(key)
	if !ok {
		// Nothing was stored, return 0
		return 0
	}
	d, ok := val.(*duration)
	if !ok {
		// edge case: corrupt stored duration
		// recover: assume nothing was stored, return 0
		return 0
	}
	return d.Get()
}

type duration struct {
	sync.Mutex
	dur time.Duration
}

// Get() safely returns the underlying value
func (d *duration) Get() time.Duration {
	d.Lock()
	defer d.Unlock()
	return d.dur
}

// Update() safely updates the underlying value, taking the larger value.
func (d *duration) Update(newDur time.Duration, eval func(oldDur, newDur time.Duration) bool) {
	d.Lock()
	defer d.Unlock()
	if eval(d.dur, newDur) {
		d.dur = newDur
	}
}

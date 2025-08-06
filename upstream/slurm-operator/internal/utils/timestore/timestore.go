// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package timestore

import (
	"sync"
	"time"
)

// TimeStore{} stores arbitrary time.Time. It allows for safe pushes and
// pops based on any arbitrary keys.
type TimeStore struct {
	// Underlying data store
	sync.Map

	// Function to resolve which time to keep when repeated push to same key
	eval func(oldTime, newTime time.Time) bool
}

// NewDurationStore() returns a time store which will evaluate which value to
// keep, when multiple are pushed to the same key, based on the eval() function.
func NewTimeStore(eval func(oldTime, newTime time.Time) bool) *TimeStore {
	return &TimeStore{eval: eval}
}

func Greater(oldTime, newTime time.Time) bool {
	return newTime.After(oldTime)
}

func Less(oldTime, newTime time.Time) bool {
	return newTime.Before(oldTime)
}

// Push() will store a time for the key. If multiple values are pushed onto
// the same key before the key is popped, then the one that returns true from
// TimeStore.eval() is kept.
func (ts *TimeStore) Push(key string, newTime time.Time) {
	newT := &timestore{t: newTime}
	val, loaded := ts.LoadOrStore(key, newT)
	if !loaded {
		return
	}
	d, ok := val.(*timestore)
	if !ok {
		// edge case: corrupt stored time
		// recover: delete key and store new time
		ts.Delete(key)
		ts.Store(key, newT)
		return
	}
	d.Update(newTime, ts.eval)
}

// Pop() will return the time stored by the key and delete the store. If no
// time was stored for that key, then 0 will be returned.
func (ts *TimeStore) Pop(key string) time.Time {
	val, ok := ts.LoadAndDelete(key)
	if !ok {
		// Nothing was stored, unit Time
		return time.Time{}
	}
	d, ok := val.(*timestore)
	if !ok {
		// edge case: corrupt stored time
		// recover: assume nothing was stored, unit Time
		return time.Time{}
	}
	return d.Get()
}

// Peek() will return the time stored by the key and *not* delete the store.
// If no time was stored for that key, then 0 will be returned.
func (ts *TimeStore) Peek(key string) time.Time {
	val, ok := ts.Load(key)
	if !ok {
		// Nothing was stored, unit Time
		return time.Time{}
	}
	d, ok := val.(*timestore)
	if !ok {
		// edge case: corrupt stored time
		// recover: assume nothing was stored, unit Time
		return time.Time{}
	}
	return d.Get()
}

type timestore struct {
	sync.Mutex
	t time.Time
}

// Get() safely returns the underlying value
func (d *timestore) Get() time.Time {
	d.Lock()
	defer d.Unlock()
	return d.t
}

// Update() safely updates the underlying value, taking the larger value.
func (d *timestore) Update(newTime time.Time, eval func(oldTime, newTime time.Time) bool) {
	d.Lock()
	defer d.Unlock()
	if eval(d.t, newTime) {
		d.t = newTime
	}
}

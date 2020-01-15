// Copyright 2019 The gVisor Authors.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.13
// +build !go1.15

// Check that syncMutex matches the standard library sync.Mutex definition.

package sync

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// TMutex is a try lock.
type TMutex struct {
	sync.Mutex
}

type syncMutex struct {
	state int32
	sema  uint32
}

const mutexLocked = 1 << iota // mutex is locked

// TryLock trys to aquire the mutex, but won't block if it can't.
func (m *TMutex) TryLock() bool {
	if atomic.CompareAndSwapInt32(&(*syncMutex)(unsafe.Pointer(m)).state, 0, mutexLocked) {
		if RaceEnabled {
			RaceAcquire(unsafe.Pointer(m))
		}
		return true
	}
	return false
}

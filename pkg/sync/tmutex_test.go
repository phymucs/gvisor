// Copyright 2019 The gVisor Authors.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sync

import (
	"testing"
)

func TestDoubleTryLock(t *testing.T) {
	var m TMutex
	if !m.TryLock() {
		t.Fatal("failed to aquire lock")
	}
	if m.TryLock() {
		t.Fatal("unexpectedly succeeded in aquiring locked mutex")
	}
}

func TestTryLockAfterLock(t *testing.T) {
	var m TMutex
	m.Lock()
	if m.TryLock() {
		t.Fatal("unexpectedly succeeded in aquiring locked mutex")
	}
}

func TestTryLockUnlock(t *testing.T) {
	var m TMutex
	if !m.TryLock() {
		t.Fatal("failed to aquire lock")
	}
	m.Unlock()
	if !m.TryLock() {
		t.Fatal("failed to aquire lock after unlock")
	}
}

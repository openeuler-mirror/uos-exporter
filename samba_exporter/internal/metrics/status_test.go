package metrics

// Copyright 2021 by tobi@backfrak.de. All
// rights reserved. Use of this source code is governed
// by a BSD-style license that can be found in the
// LICENSE file.

import (
	"sync"
	"testing"
)

var mMutext sync.Mutex = sync.Mutex{}

func TestTestPsResponse(t *testing.T) {
	mMutext.Lock()
	defer mMutext.Unlock()

	oldParmas := parms
	defer func() { parms = oldParmas }()
	responseHandler := NewPipeHandler(true, ResposePipe)

	err := testPsResponse(responseHandler, 0)
	if err != nil {
		t.Errorf("Get error '%s' but expected none", err.Error())
	}
}

func TestTestProcessResponse(t *testing.T) {
	mMutext.Lock()
	defer mMutext.Unlock()

	oldParmas := parms
	defer func() { parms = oldParmas }()
	responseHandler := NewPipeHandler(true, ResposePipe)

	err := testProcessResponse(responseHandler, 10)
	if err != nil {
		t.Errorf("Get error '%s' but expected none", err.Error())
	}
}

func TestTestLockResponse(t *testing.T) {
	mMutext.Lock()
	defer mMutext.Unlock()

	oldParmas := parms
	defer func() { parms = oldParmas }()
	responseHandler := NewPipeHandler(true, ResposePipe)

	err := testLockResponse(responseHandler, 20)
	if err != nil {
		t.Errorf("Get error '%s' but expected none", err.Error())
	}
}

func TestTestShareResponse(t *testing.T) {
	mMutext.Lock()
	defer mMutext.Unlock()

	oldParmas := parms
	defer func() { parms = oldParmas }()
	responseHandler := NewPipeHandler(true, ResposePipe)

	err := testShareResponse(responseHandler, 30)
	if err != nil {
		t.Errorf("Get error '%s' but expected none", err.Error())
	}
}

func TestHandleRequest(t *testing.T) {
	mMutext.Lock()
	defer mMutext.Unlock()

	oldParmas := parms
	defer func() { parms = oldParmas }()
	responseHandler := NewPipeHandler(true, ResposePipe)

	errNil := handleRequest(responseHandler,
		GetRequest(LOCK_REQUEST, 12),
		LOCK_REQUEST,
		func(ph *PipeHandler, i int) error { return nil },
		func(ph *PipeHandler, i int) error { return nil },
	)

	if errNil != nil {
		t.Errorf("Get error '%s' but expected none", errNil.Error())
	}

	parms.Test = true
	errNil = handleRequest(responseHandler,
		GetRequest(LOCK_REQUEST, 12),
		LOCK_REQUEST,
		func(ph *PipeHandler, i int) error { return nil },
		func(ph *PipeHandler, i int) error { return nil },
	)

	if errNil != nil {
		t.Errorf("Get error '%s' but expected none", errNil.Error())
	}

	errRequest := NewEmptyStringQueueError()
	errHandle := handleRequest(responseHandler,
		GetRequest(LOCK_REQUEST, 12),
		LOCK_REQUEST,
		func(ph *PipeHandler, i int) error { return nil },
		func(ph *PipeHandler, i int) error { return errRequest },
	)

	if errHandle != errRequest {
		t.Errorf("Got error '%s', but expected error '%s'", errHandle.Error(), errRequest.Error())
	}

	parms.Test = false
	errHandle = handleRequest(responseHandler,
		GetRequest(LOCK_REQUEST, 12),
		LOCK_REQUEST,
		func(ph *PipeHandler, i int) error { return errRequest },
		func(ph *PipeHandler, i int) error { return nil },
	)

	if errHandle != errRequest {
		t.Errorf("Got error '%s', but expected error '%s'", errHandle.Error(), errRequest.Error())
	}

}

func TestGoHandleRequestQueue(t *testing.T) {
	mMutext.Lock()
	defer mMutext.Unlock()

	oldParmas := parms
	defer func() { parms = oldParmas }()
	responseHandler := NewPipeHandler(true, ResposePipe)
	requestQueue = *NewStringQueue()
	parms.Test = true

	requestQueue.Push(GetRequest(LOCK_REQUEST, 0))
	goHandleRequestQueue(responseHandler)

	requestQueue.Push(GetRequest(SHARE_REQUEST, 1))
	goHandleRequestQueue(responseHandler)

	requestQueue.Push(GetRequest(PROCESS_REQUEST, 2))
	goHandleRequestQueue(responseHandler)

	requestQueue.Push(GetRequest(PS_REQUEST, 3))
	goHandleRequestQueue(responseHandler)

	requestQueue.Push(GetRequest("NO_REQUEST", 3))
	goHandleRequestQueue(responseHandler)

	requestQueue.Push("")
	goHandleRequestQueue(responseHandler)

}

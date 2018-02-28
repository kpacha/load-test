package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/kpacha/load-test/db"
	"github.com/kpacha/load-test/requester"
)

func TestNewExecutor_Run_contextCanceled(t *testing.T) {
	store := db.NewInMemory()
	exec := NewExecutor(store)
	p := Plan{
		Min:      1,
		Max:      10,
		Steps:    1,
		Duration: 1,
	}
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := exec.Run(ctx, p)
		if err == nil {
			t.Error("error expected")
			return
		}
		if err.Error() != "executing the plan: executing the step #1 of the plan: context canceled" {
			t.Errorf("unexpected error: %s", err.Error())
		}
	}
}

func Test_executor_Run_wrongReportFormat(t *testing.T) {
	store := db.NewInMemory()
	totalCalls := 0
	exec := executor{
		DB: store,
		RequesterFactory: func(req *http.Request) requester.Requester {
			totalCalls++
			return dummyRequester(func(ctx context.Context, c int) io.Reader {
				if totalCalls != c {
					t.Errorf("unexpected number of calls. have %d want %d", totalCalls, c)
				}
				return bytes.NewBufferString("[]")
			})
		},
	}
	p := Plan{
		Min:      1,
		Max:      10,
		Steps:    1,
		Duration: 1,
	}
	_, err := exec.Run(context.Background(), p)
	if err == nil {
		t.Error("error expected")
		return
	}
	if err.Error() != "executing the plan: decoding the results: json: cannot unmarshal array into Go value of type requester.Report" {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func Test_executor_Run_koStore(t *testing.T) {
	expectedErr := errors.New("you should expect me")
	store := erroredStore{expectedErr}

	totalCalls := 0
	exec := executor{
		DB: store,
		RequesterFactory: func(req *http.Request) requester.Requester {
			totalCalls++
			return dummyRequester(func(ctx context.Context, c int) io.Reader {
				if totalCalls != c {
					t.Errorf("unexpected number of calls. have %d want %d", totalCalls, c)
				}
				return bytes.NewBufferString("{}")
			})
		},
	}
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("building the request: %s", err.Error())
		return
	}
	p := Plan{
		Min:      1,
		Max:      10,
		Steps:    1,
		Duration: 1,
		Request:  req,
	}
	_, err = exec.Run(context.Background(), p)
	if err == nil {
		t.Error("error expected")
		return
	}
	if err.Error() != "storing the results: you should expect me" {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func Test_executor_Run_ok(t *testing.T) {
	store := db.NewInMemory()
	name := "some-name"
	totalCalls := 0
	exec := executor{
		DB: store,
		RequesterFactory: func(req *http.Request) requester.Requester {
			totalCalls++
			return dummyRequester(func(ctx context.Context, c int) io.Reader {
				if totalCalls != c {
					t.Errorf("unexpected number of calls. have %d want %d", totalCalls, c)
				}
				return bytes.NewBufferString("{}")
			})
		},
	}
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("building the request: %s", err.Error())
		return
	}
	p := Plan{
		Min:      1,
		Max:      10,
		Steps:    1,
		Duration: 1,
		Request:  req,
		Name:     name,
	}

	if _, err = exec.Run(context.Background(), p); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}
	r, err := store.Get(name)
	if err != nil {
		t.Errorf("accessing the store: %s", err.Error())
		return
	}

	results := []requester.Report{}

	if err = json.NewDecoder(r).Decode(&results); err != nil {
		t.Error(err)
		return
	}

	if len(results) != 9 {
		t.Errorf("unexpected result size: %d", len(results))
	}

}

type dummyRequester func(ctx context.Context, c int) io.Reader

func (d dummyRequester) Run(ctx context.Context, c int) io.Reader {
	return d(ctx, c)
}

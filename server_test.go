package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kpacha/load-test/db"
	"github.com/kpacha/load-test/requester"
)

func Example_urlEncode() {
	values, err := url.ParseQuery("name=test1&url=http%3A%2F%2F127.0.0.1%3A8000%2F__debug%2Fsupu%2F1&req_method=POST&min=1&max=15&steps=4&duration=10&sleep=5&headers=Accept%3A+application%2Fjson%0D%0AContent-Type%3A+application%2Fjson&body=%7B%22a%22%3A%22b%22%2C%22c%22%3Atrue%2C%22d%22%3A42%7D")
	if err != nil {
		fmt.Println("error:", err.Error())
	}
	for k, v := range values {
		if k == "headers" {
			continue
		}
		fmt.Println(k, v)
	}

	for k, v := range parseHeaders(values["headers"][0]) {
		fmt.Printf("header [%s] value: %v\n", k, v[0])
	}

	// Unordered output:
	// sleep [5]
	// body [{"a":"b","c":true,"d":42}]
	// url [http://127.0.0.1:8000/__debug/supu/1]
	// min [1]
	// steps [4]
	// duration [10]
	// name [test1]
	// req_method [POST]
	// max [15]
	// header [Accept] value: application/json
	// header [Content-Type] value: application/json
}

func Test_parseHeaders(t *testing.T) {
	h := parseHeaders("\nAccept: application/json\nContent-Type: application/json\n")
	if len(h) != 2 {
		t.Errorf("unexpected length: %d", len(h))
		return
	}
	if v := h.Get("Accept"); v != "application/json" {
		t.Errorf("unexpected Accept header: %s", v)
	}
	if v := h.Get("Content-Type"); v != "application/json" {
		t.Errorf("unexpected Content-Type header: %s", v)
	}
}

func TestNewServer_dbError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("you should expect me")
	store := erroredStore{expectedErr}

	exec := dummyExecutor(func(_ context.Context, _ Plan) ([]requester.Report, error) {
		t.Error("the executor should not been executed")
		return []requester.Report{}, nil
	})

	s, err := NewServer(gin.New(), store, exec)
	if err != nil {
		t.Error(err)
		return
	}

	for _, url := range []string{
		"/",
		"/browse/123456789",
	} {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			t.Error(err)
			return
		}
		w := httptest.NewRecorder()
		s.Engine.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusInternalServerError {
			t.Errorf("unexpected status code: %d", w.Result().StatusCode)
		}
	}
}

func TestNewServer_browseAndHome(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResult := []requester.Report{{C: 42}, {C: 123456}}
	expectedKeys := []string{
		"key1key1key1key1key1key1",
		"key2key2key2key2key2key2",
		"key3key3key3key3key3key3",
		"key4key4key4key4key4key4",
	}

	store := db.NewInMemory()
	for _, k := range expectedKeys {
		store.Set(k, bytes.NewBufferString("[]"))
	}
	store.Set("broken", bytes.NewBufferString("{}"))

	exec := dummyExecutor(func(_ context.Context, _ Plan) ([]requester.Report, error) {
		return expectedResult, nil
	})

	s, err := NewServer(gin.New(), store, exec)
	if err != nil {
		t.Error(err)
		return
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Error(err)
		return
	}
	w := httptest.NewRecorder()
	s.Engine.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", w.Result().StatusCode)
	}
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(w.Result().Body); err != nil {
		t.Error(err)
	}
	w.Result().Body.Close()

	for _, k := range expectedKeys {
		if !strings.Contains(buf.String(), k) {
			t.Errorf("%s not present in the response body", k)
		}
	}

	req, err = http.NewRequest("GET", "/browse/unknown", nil)
	if err != nil {
		t.Error(err)
		return
	}
	w = httptest.NewRecorder()
	s.Engine.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status code: %d", w.Result().StatusCode)
	}

	req, err = http.NewRequest("GET", "/browse/broken", nil)
	if err != nil {
		t.Error(err)
		return
	}
	w = httptest.NewRecorder()
	s.Engine.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("unexpected status code: %d", w.Result().StatusCode)
	}

	req, err = http.NewRequest("GET", "/browse/key1key1key1key1key1key1", nil)
	if err != nil {
		t.Error(err)
		return
	}
	w = httptest.NewRecorder()
	s.Engine.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", w.Result().StatusCode)
	}
	buf = &bytes.Buffer{}
	if _, err := buf.ReadFrom(w.Result().Body); err != nil {
		t.Error(err)
	}
	w.Result().Body.Close()

	for _, k := range expectedKeys {
		if !strings.Contains(buf.String(), k) {
			t.Errorf("%s not present in the response body", k)
		}
	}
}

func TestNewServer_createTest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedName := "subject-test"
	expectedURL := "http://some.example.com/endpoint"
	expectedMethod := "POST"
	expectedMinC := 1
	expectedMaxC := 50
	expectedSteps := 15
	expectedDuration := 1
	expectedBody := `{"a":"b","c":true,"d":42}`

	store := db.NewInMemory()

	exec := dummyExecutor(func(_ context.Context, p Plan) ([]requester.Report, error) {
		if p.Request.Method != expectedMethod {
			t.Errorf("unexpected method: %s", p.Request.Method)
		}
		if p.Request.URL.String() != expectedURL {
			t.Errorf("unexpected url: %s", p.Request.URL.String())
		}
		if v := p.Request.Header.Get("Accept"); v != "application/json" {
			t.Errorf("unexpected Accept header value: %s", v)
		}
		if v := p.Request.Header.Get("Content-Type"); v != "application/json" {
			t.Errorf("unexpected Content-Type header value: %s", v)
		}
		if p.Sleep != -1*time.Second {
			t.Errorf("unexpected sleep: %d", p.Sleep)
		}
		buf := &bytes.Buffer{}
		buf.ReadFrom(p.Request.Body)
		p.Request.Body.Close()
		if body := buf.String(); body != expectedBody {
			t.Errorf("unexpected request body: %s", body)
		}
		return []requester.Report{}, nil
	})

	s, err := NewServer(gin.New(), store, exec)
	if err != nil {
		t.Error(err)
		return
	}

	form := url.Values{}
	form.Add("name", expectedName)
	form.Add("url", expectedURL)
	form.Add("req_method", expectedMethod)
	form.Add("min", strconv.Itoa(expectedMinC))
	form.Add("max", strconv.Itoa(expectedMaxC))
	form.Add("steps", strconv.Itoa(expectedSteps))
	form.Add("duration", strconv.Itoa(expectedDuration))
	form.Add("body", expectedBody)
	form.Add("headers", "\nAccept: application/json\nContent-Type: application/json\n")

	req, err := http.NewRequest("POST", "/test", bytes.NewBufferString(form.Encode()))
	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	s.Engine.ServeHTTP(w, req)

	if w.Result().StatusCode != 301 {
		t.Errorf("unexpected status code: %d", w.Result().StatusCode)
	}
}

type dummyExecutor func(ctx context.Context, plan Plan) ([]requester.Report, error)

func (d dummyExecutor) Run(ctx context.Context, plan Plan) ([]requester.Report, error) {
	return d(ctx, plan)
}

type erroredStore struct {
	Error error
}

func (e erroredStore) Get(key string) (io.Reader, error) {
	return nil, e.Error
}
func (e erroredStore) Keys() ([]string, error) {
	return []string{}, e.Error
}

func (e erroredStore) Set(key string, r io.Reader) (int, error) {
	return -1, e.Error
}

// func Test_getRequest(t *testing.T) {

// 	headers := parseHeaders(`Accept: application/json
// Content-Type: application/json`)
// 	if len(headers) != 2 {
// 		t.Errorf("unexpected length: %d", len(h))
// 		return
// 	}
// 	if v := headers.Get("Accept"); v != "application/json" {
// 		t.Errorf("unexpected Accept header: %s", v)
// 	}
// 	if v := headers.Get("Content-Type"); v != "application/json" {
// 		t.Errorf("unexpected Content-Type header: %s", v)
// 	}
// }

package requester

import (
	"bytes"
	"context"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	hey "github.com/rakyll/hey/requester"
)

func NewCSV(req *http.Request, timeout time.Duration) Requester {
	return New(req, csvTmpl, timeout)
}

func NewJSON(req *http.Request, timeout time.Duration) Requester {
	return New(req, jsonTmpl, timeout)
}

func New(req *http.Request, tmpl string, timeout time.Duration) Requester {
	body := new(bytes.Buffer)
	if req != nil && req.Body != nil {
		body.ReadFrom(req.Body)
		req.Body.Close()
	}
	return requester{
		Request: req,
		Body:    body.Bytes(),
		N:       math.MaxInt,
		Timeout: timeout,
		Tmpl:    tmpl,
	}
}

type Requester interface {
	Run(ctx context.Context, c int) io.Reader
}

type requester struct {
	Request *http.Request
	Body    []byte
	N       int
	Timeout time.Duration
	Tmpl    string
}

func (r requester) Run(ctx context.Context, c int) io.Reader {
	buf := new(bytes.Buffer)

	work := hey.Work{
		N:           r.N,
		C:           c,
		Timeout:     int(r.Timeout / time.Second),
		RequestBody: r.Body,
		Request:     r.Request,
		Output:      r.Tmpl,
		Writer:      buf,
	}
	if work.Output == "" {
		work.Output = csvTmpl
	}

	localCtx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	go func(localCtx context.Context, cancelWorkFunc func()) {
		<-localCtx.Done()
		cancelWorkFunc()
	}(localCtx, work.Stop)

	log.Println("starting the load test")
	work.Run()
	log.Println("load test ended")

	return buf
}

var (
	csvTmpl = `{{ $connLats := .ConnLats }}{{ $dnsLats := .DnsLats }}{{ $dnsLats := .DnsLats }}{{ $reqLats := .ReqLats }}{{ $delayLats := .DelayLats }}{{ $resLats := .ResLats }}
response-time,DNS+dialup,DNS,Request-write,Response-delay,Response-read{{ range $i, $v := .Lats }}
{{ $v }},{{ (index $connLats $i) }},{{ (index $dnsLats $i) }},{{ (index $reqLats $i) }},{{ (index $delayLats $i) }},{{ (index $resLats $i) }}{{ end }}`

	jsonTmpl = `{{ jsonify . }}`
)

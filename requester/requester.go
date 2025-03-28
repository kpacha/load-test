package requester

import (
	"bytes"
	"context"
	"io"
	"net/http"

	hey "github.com/rakyll/hey/requester"
)

func NewCSV(req *http.Request) Requester {
	return New(req, csvTmpl)
}

func NewJSON(req *http.Request) Requester {
	return New(req, jsonTmpl)
}

func New(req *http.Request, tmpl string) Requester {
	body := new(bytes.Buffer)
	if req != nil && req.Body != nil {
		body.ReadFrom(req.Body)
		req.Body.Close()
	}
	return requester{
		Request: req,
		Body:    body.Bytes(),
		N:       1000000,
		Timeout: 5,
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
	Timeout int
	Tmpl    string
}

func (r requester) Run(ctx context.Context, c int) io.Reader {
	buf := new(bytes.Buffer)

	work := hey.Work{
		N:           r.N,
		C:           c,
		Timeout:     r.Timeout,
		RequestBody: r.Body,
		Request:     r.Request,
		Output:      r.Tmpl,
		Writer:      buf,
	}
	if work.Output == "" {
		work.Output = csvTmpl
	}

	localCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func(localCtx context.Context, work hey.Work) {
		<-localCtx.Done()
		work.Stop()
	}(localCtx, work)

	work.Run()

	return buf
}

var (
	csvTmpl = `{{ $connLats := .ConnLats }}{{ $dnsLats := .DnsLats }}{{ $dnsLats := .DnsLats }}{{ $reqLats := .ReqLats }}{{ $delayLats := .DelayLats }}{{ $resLats := .ResLats }}
response-time,DNS+dialup,DNS,Request-write,Response-delay,Response-read{{ range $i, $v := .Lats }}
{{ $v }},{{ (index $connLats $i) }},{{ (index $dnsLats $i) }},{{ (index $reqLats $i) }},{{ (index $delayLats $i) }},{{ (index $resLats $i) }}{{ end }}`

	jsonTmpl = `{{ jsonify . }}`
)

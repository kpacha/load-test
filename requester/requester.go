package requester

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/rakyll/hey/requester"
)

func NewCSV(req *http.Request) Requester {
	return New(req, csvTmpl)
}

func NewJSON(req *http.Request) Requester {
	return New(req, jsonTmpl)
}

func New(req *http.Request, tmpl string) Requester {
	return Requester{
		Request: req,
		N:       10000,
		Timeout: 5,
		Tmpl:    tmpl,
	}
}

type Requester struct {
	Request *http.Request
	N       int
	Timeout int
	Tmpl    string
}

func (r Requester) Run(ctx context.Context, c int) io.Reader {
	buf := bytes.NewBuffer([]byte{})
	work := requester.Work{
		N:           r.N,
		C:           c,
		Timeout:     r.Timeout,
		RequestBody: make([]byte, 0),
		Request:     r.Request,
		Output:      r.Tmpl,
		Writer:      buf,
	}
	if work.Output == "" {
		work.Output = csvTmpl
	}

	localCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func(localCtx context.Context, work requester.Work) {
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

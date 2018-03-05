package requester

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"io/ioutil"

	"fmt"

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
	var (
		body []byte
		err  error
	)

	defer r.Request.Body.Close()

	if r.Request.Body != nil {
		body, err = ioutil.ReadAll(r.Request.Body)
		if err != nil {
			fmt.Println("request body reading error:", err.Error())
		}
	}

	r.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	work := requester.Work{
		N:           r.N,
		C:           c,
		Timeout:     r.Timeout,
		RequestBody: body,
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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kpacha/load-test/db"
	"github.com/kpacha/load-test/requester"
)

type Executor interface {
	Run(ctx context.Context, plan Plan) error
}

func NewExecutor(store db.DB) Executor {
	return &executor{store}
}

type executor struct {
	DB db.DB
}

func (e *executor) Run(ctx context.Context, plan Plan) error {
	work.Lock()
	defer work.Unlock()
	report, err := plan.Run(ctx)
	if err != nil {
		return err
	}

	data := &bytes.Buffer{}
	if err := json.NewEncoder(data).Encode(report); err != nil {
		return err
	}

	_, err = e.DB.Set(plan.Name, data)
	return err
}

var (
	work = &sync.Mutex{}
)

type Plan struct {
	Name     string
	Min      int
	Max      int
	Steps    int
	Request  *http.Request
	Duration time.Duration
	Sleep    time.Duration
}

func (e Plan) String() string {
	return fmt.Sprintf("C: %d [%d-%d], Duration: %s", e.Steps, e.Min, e.Max, e.Duration.String())
}

func (e Plan) Run(ctx context.Context) ([]requester.Report, error) {
	results := []requester.Report{}

	for i := e.Min; i < e.Max; i += e.Steps {
		time.Sleep(e.Sleep)
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
		log.Printf("runing with C=%d ...\n", i)

		localCtx := ctx
		if e.Duration > 0 {
			lctx, localCancel := context.WithTimeout(ctx, e.Duration)
			defer localCancel()
			localCtx = lctx
		}

		r := requester.NewJSON(e.Request).Run(localCtx, i)

		report := requester.Report{}
		if err := json.NewDecoder(r).Decode(&report); err != nil {
			return results, err
		}
		report.C = i
		report.URL = e.Request.URL.String()

		results = append(results, report)
	}

	return results, nil
}

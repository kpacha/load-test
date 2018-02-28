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

type Executor interface {
	Run(ctx context.Context, plan Plan) ([]requester.Report, error)
}

func NewExecutor(store db.DB) Executor {
	return &executor{store}
}

type executor struct {
	DB db.DB
}

func (e *executor) Run(ctx context.Context, plan Plan) ([]requester.Report, error) {
	report, err := e.executePlan(ctx, plan)
	if err != nil {
		return report, fmt.Errorf("executing the plan: %s", err.Error())
	}

	data := &bytes.Buffer{}
	if err := json.NewEncoder(data).Encode(report); err != nil {
		return report, fmt.Errorf("encoding the report: %s", err.Error())
	}

	if _, err = e.DB.Set(plan.Name, data); err != nil {
		return report, fmt.Errorf("storing the results: %s", err.Error())
	}
	return report, nil
}

func (e *executor) executePlan(ctx context.Context, plan Plan) ([]requester.Report, error) {
	work.Lock()
	defer work.Unlock()

	results := []requester.Report{}

	for i := plan.Min; i < plan.Max; i += plan.Steps {
		fmt.Println("waiting before the next batch...")
		time.Sleep(plan.Sleep)
		select {
		case <-ctx.Done():
			return results, fmt.Errorf("executing the step #%d of the plan: %s", i, ctx.Err())
		default:
		}
		log.Printf("runing with C=%d ...\n", i)

		localCtx := ctx
		if plan.Duration > 0 {
			lctx, localCancel := context.WithTimeout(ctx, plan.Duration)
			defer localCancel()
			localCtx = lctx
		}

		r := requester.NewJSON(plan.Request).Run(localCtx, i)

		report := requester.Report{}
		if err := json.NewDecoder(r).Decode(&report); err != nil {
			return results, fmt.Errorf("decoding the results: %s", err.Error())
		}
		report.C = i
		report.URL = plan.Request.URL.String()

		results = append(results, report)
	}

	return results, nil
}

var work = &sync.Mutex{}

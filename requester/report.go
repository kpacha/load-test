package requester

import (
	"time"

	hey "github.com/rakyll/hey/requester"
)

type Report struct {
	hey.Report
	C             int
	URL           string
	pdf           Sequence
	pdfCalculated bool
	cdf           Sequence
	cdfCalculated bool
}

type Sequence struct {
	Labels []time.Duration
	Values []float64
}

func (r *Report) PDF() Sequence {
	if r.pdfCalculated {
		return r.pdf
	}
	pdf := Sequence{
		Labels: make([]time.Duration, len(r.Report.Histogram)),
		Values: make([]float64, len(r.Report.Histogram)),
	}
	for i, ld := range r.Report.Histogram {
		pdf.Labels[i] = time.Duration(int64(ld.Mark * float64(time.Second)))
		pdf.Values[i] = ld.Frequency
	}
	r.pdfCalculated = true
	r.pdf = pdf
	return pdf
}

func (r *Report) CDF() Sequence {
	if r.cdfCalculated {
		return r.cdf
	}
	pdf := r.PDF()
	cdf := Sequence{
		Labels: pdf.Labels,
		Values: make([]float64, len(pdf.Values)),
	}
	for i, p := range pdf.Values {
		if i == 0 {
			cdf.Values[i] = p
			continue
		}
		cdf.Values[i] = p + cdf.Values[i-1]
	}
	r.cdfCalculated = true
	r.cdf = cdf
	return cdf
}

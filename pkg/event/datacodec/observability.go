package datacodec

import (
	"github.com/cloudevents/sdk-go/pkg/observability"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	// LatencyMs measures the latency in milliseconds for the CloudEvents generic
	// codec data methods.
	LatencyMs = stats.Float64("cloudevents.io/sdk-go/datacodec/latency", "The latency in milliseconds for the CloudEvents generic data codec methods.", "ms")
)

var (
	// LatencyView is an OpenCensus view that shows data codec method latency.
	LatencyView = &view.View{
		Name:        "datacodec/latency",
		Measure:     LatencyMs,
		Description: "The distribution of latency inside of the generic data codec for CloudEvents.",
		Aggregation: view.Distribution(0, .01, .1, 1, 10, 100, 1000, 10000),
		TagKeys:     observability.LatencyTags(),
	}
)

type observed int32

// Adheres to Observable
var _ observability.Observable = observed(0)

const (
	reportEncode observed = iota
	reportDecode
)

// MethodName implements Observable.MethodName
func (o observed) MethodName() string {
	switch o {
	case reportEncode:
		return "encode"
	case reportDecode:
		return "decode"
	default:
		return "unknown"
	}
}

// LatencyMs implements Observable.LatencyMs
func (o observed) LatencyMs() *stats.Float64Measure {
	return LatencyMs
}

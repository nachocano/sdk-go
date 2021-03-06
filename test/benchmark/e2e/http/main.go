package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math/rand"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/pkg/event"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/cloudevents/sdk-go/pkg/binding"
	"github.com/cloudevents/sdk-go/pkg/binding/buffering"
	"github.com/cloudevents/sdk-go/pkg/transport"
	http "github.com/cloudevents/sdk-go/pkg/transport/http"
	"github.com/cloudevents/sdk-go/test/benchmark/e2e"
)

var letters = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func fillRandom(buf []byte, r *rand.Rand) {
	for i := 0; i < cap(buf); i++ {
		buf[i] = letters[r.Intn(len(letters))]
	}
}

// Avoid DCE
var W *httptest.ResponseRecorder
var R *nethttp.Request

func benchmarkBaseline(cases []e2e.BenchmarkCase, requestFactory func([]byte) *nethttp.Request) e2e.BenchmarkResults {
	var results e2e.BenchmarkResults
	r := rand.New(rand.NewSource(time.Now().Unix()))

	for _, c := range cases {
		if c.OutputSenders > 1 {
			// It doesn't make sense for this test
			continue
		}
		fmt.Printf("%+v\n", c)

		buffer := make([]byte, c.PayloadSize)
		fillRandom(buffer, r)

		result := testing.Benchmark(func(b *testing.B) {
			b.SetParallelism(c.Parallelism)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					W = httptest.NewRecorder()
					R = requestFactory(buffer)
				}
			})
		})
		results = append(results, e2e.BenchmarkResult{BenchmarkCase: c, BenchmarkResult: result})
	}

	return results
}

func pipeLoopDirect(r *http.Receiver, sendCtx context.Context, endCtx context.Context, opts ...http.SenderOptionFunc) {
	s := MockedSender(opts...)
	var err error
	var m binding.Message
	for err != io.EOF {
		select {
		case <-endCtx.Done():
			return
		default:
			m, err = r.Receive(endCtx)
			if err != nil || m == nil {
				continue
			}
			_ = s.Send(sendCtx, m)
		}
	}
}

func pipeLoopMulti(r *http.Receiver, sendCtx context.Context, endCtx context.Context, outputSenders int, opts ...http.SenderOptionFunc) {
	s := MockedSender(opts...)
	var err error
	var m binding.Message
	for err != io.EOF {
		select {
		case <-endCtx.Done():
			return
		default:
			m, err = r.Receive(endCtx)
			if err != nil {
				continue
			}
			copiedMessage, err := buffering.BufferMessage(context.TODO(), m, nil)
			if err != nil {
				continue
			}
			outputMessage := buffering.WithAcksBeforeFinish(copiedMessage, outputSenders)
			for i := 0; i < outputSenders; i++ {
				go func(m binding.Message) {
					_ = s.Send(sendCtx, outputMessage)
				}(outputMessage)
			}
		}
	}
}

func benchmarkReceiverSender(cases []e2e.BenchmarkCase, requestFactory func([]byte) *nethttp.Request, contextDecorator func(context.Context) context.Context) e2e.BenchmarkResults {
	var results e2e.BenchmarkResults
	random := rand.New(rand.NewSource(time.Now().Unix()))

	for _, c := range cases {
		fmt.Printf("%+v\n", c)

		ctx, cancel := context.WithCancel(context.TODO())
		receiver := http.NewReceiver()

		// Spawn dispatchers
		for i := 0; i < c.Parallelism; i++ {
			if c.OutputSenders == 1 {
				go pipeLoopDirect(receiver, contextDecorator(context.TODO()), ctx)
			} else {
				go pipeLoopMulti(receiver, contextDecorator(context.TODO()), ctx, c.OutputSenders)
			}
		}

		buffer := make([]byte, c.PayloadSize)
		fillRandom(buffer, random)
		runtime.GC()

		result := testing.Benchmark(func(b *testing.B) {
			b.SetParallelism(c.Parallelism)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					w := httptest.NewRecorder()
					receiver.ServeHTTP(w, requestFactory(buffer))
				}
			})
		})
		results = append(results, e2e.BenchmarkResult{BenchmarkCase: c, BenchmarkResult: result})

		cancel()
		runtime.GC()
	}

	return results
}

type benchDelivery struct {
	fn transport.DeliveryFunc
}

func (b *benchDelivery) Delivery(ctx context.Context, e event.Event, er *event.EventResponse) error {
	return b.fn(ctx, e, er)
}

func dispatchReceiver(clients []cloudevents.Client, outputSenders int) transport.Delivery {
	return &benchDelivery{fn: func(ctx context.Context, e cloudevents.Event, er *cloudevents.EventResponse) error {
		var wg sync.WaitGroup
		for i := 0; i < outputSenders; i++ {
			wg.Add(1)
			go func(client cloudevents.Client) {
				_ = client.Send(ctx, e)
				wg.Done()
			}(clients[i])
		}
		wg.Wait()
		er.RespondWith(200, nil)
		return nil
	}}
}

func benchmarkClient(cases []e2e.BenchmarkCase, requestFactory func([]byte) *nethttp.Request) e2e.BenchmarkResults {
	var results e2e.BenchmarkResults
	random := rand.New(rand.NewSource(time.Now().Unix()))

	for _, c := range cases {
		fmt.Printf("%+v\n", c)

		_, mockedReceiverTransport := MockedClient()

		senderClients := make([]cloudevents.Client, c.OutputSenders)
		for i := 0; i < c.OutputSenders; i++ {
			senderClients[i], _ = MockedClient()
		}

		mockedReceiverTransport.SetDelivery(dispatchReceiver(senderClients, c.OutputSenders))

		buffer := make([]byte, c.PayloadSize)
		fillRandom(buffer, random)
		runtime.GC()

		result := testing.Benchmark(func(b *testing.B) {
			b.SetParallelism(c.Parallelism)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					w := httptest.NewRecorder()
					mockedReceiverTransport.ServeHTTP(w, requestFactory(buffer))
				}
			})
		})
		results = append(results, e2e.BenchmarkResult{BenchmarkCase: c, BenchmarkResult: result})

		runtime.GC()
	}

	return results
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var bench = flag.String(
	"bench",
	"baseline-binary",
	"[baseline-structured, baseline-binary, binding-structured-to-structured, binding-structured-to-binary, binding-binary-to-structured, binding-binary-to-binary, client-binary, client-structured]",
)
var out = flag.String("out", "out.csv", "Output file")
var maxPayloadKb = flag.Int("max-payload", 32, "Max payload size in kb")
var maxParallelism = flag.Int("max-parallelism", runtime.NumCPU()*2, "Max parallelism")
var maxOutputSenders = flag.Int("max-output-senders", 1, "Max output senders")

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, _ := os.Create(*cpuprofile)
		_ = pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	benchmarkCases := e2e.GenerateAllBenchmarkCases(
		1024,
		1024*(*maxPayloadKb),
		1,
		*maxParallelism,
		1,
		*maxOutputSenders, //TODO to be increased when receiver-sender will support multi senders
	)

	var results e2e.BenchmarkResults

	fmt.Printf("--- Starting benchmark %s ---\n", *bench)

	switch *bench {
	case "baseline-structured":
		results = benchmarkBaseline(benchmarkCases, MockedStructuredRequest)
		break
	case "baseline-binary":
		results = benchmarkBaseline(benchmarkCases, MockedBinaryRequest)
		break
	case "binding-structured-to-structured":
		results = benchmarkReceiverSender(benchmarkCases, MockedStructuredRequest, binding.WithForceStructured)
		break
	case "binding-structured-to-binary":
		results = benchmarkReceiverSender(benchmarkCases, MockedStructuredRequest, binding.WithForceBinary)
		break
	case "binding-binary-to-structured":
		results = benchmarkReceiverSender(benchmarkCases, MockedBinaryRequest, binding.WithForceStructured)
		break
	case "binding-binary-to-binary":
		results = benchmarkReceiverSender(benchmarkCases, MockedBinaryRequest, binding.WithForceBinary)
		break
	case "client-binary":
		results = benchmarkClient(benchmarkCases, MockedBinaryRequest)
		break
	case "client-structured":
		results = benchmarkClient(benchmarkCases, MockedStructuredRequest)
		break
	default:
		panic("Wrong bench flag")
	}

	pprof.StopCPUProfile()

	if *memprofile != "" {
		f, _ := os.Create(*memprofile)
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		_ = pprof.WriteHeapProfile(f)
	}

	f, err := os.Create(*out)
	if err != nil {
		panic(fmt.Sprintf("Cannot open file %s: %v", *out, err))
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	err = results.WriteToCsv(writer)
	if err != nil {
		panic(err)
	}
}

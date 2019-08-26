package http_test

import (
	"context"
	nethttp "net/http"
	"sort"
	"testing"

	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTransportContext(t *testing.T) {
	testCases := map[string]struct {
		t    http.TransportContext
		ctx  context.Context
		want http.TransportContext
	}{
		"nil context": {},
		"nil context, set transport context": {
			t: http.TransportContext{
				Host:   "unit test",
				Method: "unit test",
			},
			want: http.TransportContext{
				Host:   "unit test",
				Method: "unit test",
			},
		},
		"todo context, set transport context": {
			ctx: context.TODO(),
			t: http.TransportContext{
				Host:   "unit test",
				Method: "unit test",
			},
			want: http.TransportContext{
				Host:   "unit test",
				Method: "unit test",
			},
		},
		"bad transport context": {
			ctx: context.TODO(),
		},
		"already set transport context": {
			ctx: http.WithTransportContext(context.TODO(),
				http.TransportContext{
					Host:   "existing test",
					Method: "exiting test",
				}),
			t: http.TransportContext{
				Host:   "unit test",
				Method: "unit test",
			},
			want: http.TransportContext{
				Host:   "unit test",
				Method: "unit test",
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			ctx := http.WithTransportContext(tc.ctx, tc.t)

			got := http.TransportContextFrom(ctx)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected (-want, +got) = %v", diff)
			}
		})
	}
}

func TestTransportResponseContext(t *testing.T) {
	testCases := map[string]struct {
		t    http.TransportResponseContext
		ctx  context.Context
		want http.TransportResponseContext
	}{
		"nil context": {},
		"nil context, set transport response context": {
			t: http.TransportResponseContext{
				StatusCode: nethttp.StatusOK,
			},
			want: http.TransportResponseContext{
				StatusCode: nethttp.StatusOK,
			},
		},
		"todo context, set transport response context": {
			ctx: context.TODO(),
			t: http.TransportResponseContext{
				StatusCode: nethttp.StatusOK,
			},
			want: http.TransportResponseContext{
				StatusCode: nethttp.StatusOK,
			},
		},
		"bad transport response context": {
			ctx: context.TODO(),
		},
		"already set transport response context": {
			ctx: http.WithTransportResponseContext(context.TODO(),
				http.TransportResponseContext{
					StatusCode: nethttp.StatusOK,
				}),
			t: http.TransportResponseContext{
				StatusCode: nethttp.StatusOK,
			},
			want: http.TransportResponseContext{
				StatusCode: nethttp.StatusOK,
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			ctx := http.WithTransportResponseContext(tc.ctx, tc.t)

			got := http.TransportResponseContextFrom(ctx)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected (-want, +got) = %v", diff)
			}
		})
	}
}

func TestNewTransportContext(t *testing.T) {
	testCases := map[string]struct {
		r       *nethttp.Request
		want    http.TransportContext
		wantStr string
	}{
		"nil request": {
			want: http.TransportContext{},
			wantStr: `Transport Context,
  nil
`,
		},
		"full request": {
			r: &nethttp.Request{
				Host:       "unit test host",
				Method:     "unit test method",
				RequestURI: "unit test uri",
				Header: func() nethttp.Header {
					h := nethttp.Header{}
					h.Set("unit", "test header")
					return h
				}(),
			},
			want: http.TransportContext{
				Host:   "unit test host",
				Method: "unit test method",
				URI:    "unit test uri",
				Header: func() nethttp.Header {
					h := nethttp.Header{}
					h.Set("unit", "test header")
					return h
				}(),
			},
			wantStr: `Transport Context,
  URI: unit test uri
  Host: unit test host
  Method: unit test method
  Header:
    Unit: test header
`,
		},
		"no headers request": {
			r: &nethttp.Request{
				Host:       "unit test host",
				Method:     "unit test method",
				RequestURI: "unit test uri",
			},
			want: http.TransportContext{
				Host:   "unit test host",
				Method: "unit test method",
				URI:    "unit test uri",
			},
			wantStr: `Transport Context,
  URI: unit test uri
  Host: unit test host
  Method: unit test method
`,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			got := http.NewTransportContext(tc.r)

			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(http.TransportContext{}, "IgnoreHeaderPrefixes")); diff != "" {
				t.Errorf("unexpected (-want, +got) = %v", diff)
			}

			if tc.wantStr != "" {
				gotStr := got.String()

				if diff := cmp.Diff(tc.wantStr, gotStr); diff != "" {
					t.Errorf("unexpected (-want, +got) = %v", diff)
				}
			}
		})
	}
}

func TestNewTransportResponseContext(t *testing.T) {
	testCases := map[string]struct {
		r       *nethttp.Response
		want    http.TransportResponseContext
		wantStr string
	}{
		"nil response": {
			want: http.TransportResponseContext{},
			wantStr: `Transport Response Context,
  nil
`,
		},
		"full response": {
			r: &nethttp.Response{
				Header: func() nethttp.Header {
					h := nethttp.Header{}
					h.Set("unit", "test header")
					return h
				}(),
				StatusCode: nethttp.StatusOK,
			},
			want: http.TransportResponseContext{
				Header: func() nethttp.Header {
					h := nethttp.Header{}
					h.Set("unit", "test header")
					return h
				}(),
				StatusCode: nethttp.StatusOK,
			},
			wantStr: `Transport Response Context,
  StatusCode: 200
  Header:
    Unit: test header
`,
		},
		"no headers response": {
			r: &nethttp.Response{
				StatusCode: nethttp.StatusOK,
			},
			want: http.TransportResponseContext{
				StatusCode: nethttp.StatusOK,
			},
			wantStr: `Transport Response Context,
  StatusCode: 200
`,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			got := http.NewTransportResponseContext(tc.r)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected (-want, +got) = %v", diff)
			}

			if tc.wantStr != "" {
				gotStr := got.String()

				if diff := cmp.Diff(tc.wantStr, gotStr); diff != "" {
					t.Errorf("unexpected (-want, +got) = %v", diff)
				}
			}
		})
	}
}

func TestAttendToHeader(t *testing.T) {
	testCases := map[string]struct {
		header nethttp.Header
		ignore []string
		want   []string
	}{
		"nil": {},
		"no new ignore": {
			header: func() nethttp.Header {
				h := nethttp.Header{}
				h.Set("unit", "test header")
				h.Set("testing", "header unit")
				return h
			}(),
			want: []string{"Unit", "Testing"},
		},
		"with ignore": {
			header: func() nethttp.Header {
				h := nethttp.Header{}
				h.Set("unit", "test header")
				h.Set("testing", "header unit")
				return h
			}(),
			ignore: []string{"test"},
			want:   []string{"Unit"},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			tx := http.NewTransportContext(&nethttp.Request{
				Header: tc.header,
			})

			tx.AddIgnoreHeaderPrefix(tc.ignore...)

			got := tx.AttendToHeaders()

			// Sort to make the test work.
			sort.Strings(got)
			sort.Strings(tc.want)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected (-want, +got) = %v", diff)
			}
		})
	}
}

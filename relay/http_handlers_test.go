package relay

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vente-privee/influxdb-relay/config"
)

var (
	promBody   = Body{}
	influxBody = Body{}

	basicPingWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"X-Influxdb-Version": []string{"relay"},
		},
		code: 204,
	}
	teapotPingWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"X-Influxdb-Version": []string{"relay"},
			"Content-Length":     []string{"0"},
		},
		code: http.StatusTeapot,
	}
	wrongMethodPingWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	basicStatusWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"\\\"status\\\": {\\\"test\\\":{\\\"location\\\":\\\"\\\"}}\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"44"},
		},
		code: http.StatusOK,
	}
	wrongMethodStatusWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	wrongMethodPromWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Allow":          []string{"POST"},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	wrongBackendPromWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Allow":          []string{"POST"},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	BackendDownPromWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Allow":          []string{"POST"},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	BackendUpPromWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header:   http.Header{},
		code:     http.StatusNoContent,
	}
	BackendUpPromError400Writer = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"Content-Length": []string{"0"},
		},
		code: http.StatusBadRequest,
	}
	BackendUpPromError500Writer = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	wrongMethodInfluxWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"" + http.StatusText(http.StatusMethodNotAllowed) + "\"")),
		header: http.Header{
			"Allow":          []string{"POST"},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{fmt.Sprintf("%d", len(http.StatusText(http.StatusMethodNotAllowed))+2)},
		},
		code: http.StatusMethodNotAllowed,
	}
	wrongBackendInfluxWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Allow":          []string{"POST"},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	BackendDownInfluxWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Allow":          []string{"POST"},
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	BackendUpInfluxWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header:   http.Header{},
		code:     http.StatusNoContent,
	}
	BackendUpInfluxError400Writer = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header: http.Header{
			"Content-Length": []string{"0"},
		},
		code: http.StatusBadRequest,
	}
	BackendUpInfluxError500Writer = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to write points\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusServiceUnavailable,
	}
	InfluxFailParsePointWriter = &ResponseWriter{
		writeBuf: bytes.NewBuffer([]byte("\"unable to parse points\"")),
		header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"24"},
		},
		code: http.StatusBadRequest,
	}
	InfluxParsePointWriter = &ResponseWriter{
		writeBuf: &bytes.Buffer{},
		header:   http.Header{},
		code:     http.StatusNoContent,
	}
)

var (
	configPingTeapot = config.HTTPConfig{
		DefaultPingResponse: http.StatusTeapot,
	}
)

func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stderr)
	return buf.String()
}

func TestHandlePingSimple(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	r, err := http.NewRequest("GET", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handlePing(w, r)
	WriterTest(t, basicPingWriter, w)
}

func TestHandlePingTeapot(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, configPingTeapot, false)
	r, err := http.NewRequest("GET", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handlePing(w, r)
	WriterTest(t, teapotPingWriter, w)
}

func TestHandlePingWrongMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, configPingTeapot, false)
	r, err := http.NewRequest("OPTIONS", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handlePing(w, r)
	WriterTest(t, wrongMethodPingWriter, w)
}

func TestHandleStatusSimple(t *testing.T) {
	defer resetWriter()
	cfgOut := config.HTTPOutputConfig{Name: "test", InputType: "influxdb"}
	h := createHTTP(t, config.HTTPConfig{}, false)
	r, err := http.NewRequest("GET", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOut)
	h.backends = append(h.backends, b)
	h.handleStatus(w, r)
	WriterTest(t, basicStatusWriter, w)
	h.backends = h.backends[:0]
}

func TestHandleStatusWrongMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	r, err := http.NewRequest("OPTIONS", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleStatus(w, r)
	WriterTest(t, wrongMethodStatusWriter, w)
}

func TestHandlePromWrongMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	r, err := http.NewRequest("WRONG", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleProm(w, r)
	WriterTest(t, wrongMethodPromWriter, w)
}

func TestHandlePromWrongBackend(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("OPTIONS", "influxdb", promBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleProm(w, r)
	WriterTest(t, wrongBackendPromWriter, w)
}

func TestHandlePromBackendDown(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus"}
	cfgOutInflux := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb"}
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("OPTIONS", "influxdb", promBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutInflux)
	b2, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.backends = append(h.backends, b2)
	output := captureOutput(func() {
		h.handleProm(w, r)
	})
	output = output[20:]
	assert.Equal(t, "Problem posting to relay \"http://\" backend \"test_prometheus\": Post : unsupported protocol scheme \"\"\n", output)
	WriterTest(t, BackendDownPromWriter, w)
	h.backends = h.backends[:0]
}

func TestHandlePromBackendUp(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/prom" {
			res.WriteHeader(http.StatusOK)
		}
	}))
	defer testServer.Close()
	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus", Location: testServer.URL + "/prom"}
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("POST", testServer.URL, promBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleProm(w, r)
	WriterTest(t, BackendUpPromWriter, w)
	h.backends = h.backends[:0]
}

func TestHandlePromBackendUpError400(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusBadRequest)
	}))
	defer testServer.Close()
	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus", Location: testServer.URL + "/prom"}
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("POST", testServer.URL, promBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleProm(w, r)
	WriterTest(t, BackendUpPromError400Writer, w)
	h.backends = h.backends[:0]
}

func TestHandlePromBackendUpError500(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()
	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus", Location: testServer.URL + "/prom"}
	promBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("POST", testServer.URL, promBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	output := captureOutput(func() {
		h.handleProm(w, r)
	})
	output = output[20:]
	assert.Equal(t, "5xx response for relay \"http://\" backend \"test_prometheus\": 500\n", output)
	WriterTest(t, BackendUpPromError500Writer, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxWrongMethod(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	r, err := http.NewRequest("WRONG", "influxdb", emptyBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleStandard(w, r)
	WriterTest(t, wrongMethodInfluxWriter, w)
}

func TestHandleInfluxWrongBackend(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("OPTIONS", "influxdb", influxBody)
	if err != nil {
		t.Fatal(err)
	}

	h.handleStandard(w, r)
	WriterTest(t, wrongBackendInfluxWriter, w)
}

func TestHandleInfluxBackendDown(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	cfgOutProm := config.HTTPOutputConfig{Name: "test_prometheus", InputType: "prometheus"}
	cfgOutInflux := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("OPTIONS", "influxdb", influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutInflux)
	b2, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.backends = append(h.backends, b2)
	output := captureOutput(func() {
		h.handleStandard(w, r)
	})
	output = output[20:]
	assert.Equal(t, "Problem posting to relay \"http://\" backend \"test_influx\": Post : unsupported protocol scheme \"\"\n", output)
	WriterTest(t, BackendDownInfluxWriter, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxBackendUp(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/influxdb" {
			res.WriteHeader(http.StatusOK)
		}
	}))
	defer testServer.Close()
	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: testServer.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("POST", testServer.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleStandard(w, r)
	WriterTest(t, BackendUpInfluxWriter, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxBackendUpError400(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusBadRequest)
	}))
	defer testServer.Close()
	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: testServer.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("POST", testServer.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleStandard(w, r)
	WriterTest(t, BackendUpInfluxError400Writer, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxBackendUpError500(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()
	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: testServer.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte{})
	r, err := http.NewRequest("POST", testServer.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	output := captureOutput(func() {
		h.handleStandard(w, r)
	})
	output = output[20:]
	assert.Equal(t, "5xx response for relay \"http://\" backend \"test_influx\": 500\n", output)
	WriterTest(t, BackendUpInfluxError500Writer, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxFailParsePoint(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/influxdb" {
			res.WriteHeader(http.StatusOK)
		}
	}))
	defer testServer.Close()
	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: testServer.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte("Some Bug"))
	r, err := http.NewRequest("POST", testServer.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleStandard(w, r)
	WriterTest(t, InfluxFailParsePointWriter, w)
	h.backends = h.backends[:0]
}

func TestHandleInfluxParsePoint(t *testing.T) {
	defer resetWriter()
	h := createHTTP(t, config.HTTPConfig{}, false)
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/influxdb" {
			res.WriteHeader(http.StatusOK)
		}
	}))
	defer testServer.Close()
	cfgOutProm := config.HTTPOutputConfig{Name: "test_influx", InputType: "influxdb", Location: testServer.URL + "/influxdb"}
	influxBody.buf = bytes.NewBuffer([]byte("cpu_load_short,host=server01,region=us-west value=0.64 1434055562000000000"))
	r, err := http.NewRequest("POST", testServer.URL, influxBody)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newHTTPBackend(&cfgOutProm)
	h.backends = append(h.backends, b)
	h.handleStandard(w, r)
	WriterTest(t, InfluxParsePointWriter, w)
	h.backends = h.backends[:0]
}

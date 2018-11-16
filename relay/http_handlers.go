package relay

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/influxdata/influxdb/models"
	"github.com/vente-privee/influxdb-relay/config"
)

func (h *HTTP) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		st := make(map[string]map[string]string)

		for _, b := range h.backends {
			st[b.name] = b.poster.getStats()
		}

		j, _ := json.Marshal(st)

		jsonResponse(w, response{http.StatusOK, fmt.Sprintf("\"status\": %s", string(j))})
	} else {
		jsonResponse(w, response{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)})
		return
	}
}

func (h *HTTP) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		for key, value := range h.pingResponseHeaders {
			w.Header().Add(key, value)
		}
		w.WriteHeader(h.pingResponseCode)
	} else {
		jsonResponse(w, response{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)})
		return
	}
}

func (h *HTTP) handleAdmin(w http.ResponseWriter, r *http.Request) {
	// Client to perform the raw queries
	client := http.Client{}

	if r.Method == http.MethodPost {
		// Responses
		var responses = make(chan *http.Response, len(h.backends))

		// Associated waitgroup
		var wg sync.WaitGroup
		wg.Add(len(h.backends))

		// Iterate over all backends
		for _, b := range h.backends {
			b := b

			if b.admin == "" {
				// Empty query, skip backend
				wg.Done()
				continue
			}

			go func() {
				defer wg.Done()

				// Create new request
				// Update location according to backend
				// Forward body
				req, err := http.NewRequest("POST", b.admin, r.Body)
				if err != nil {
					log.Printf("Problem posting to relay %q backend %q: could not prepare request: %v", h.Name(), b.name, err)
					responses <- &http.Response{}
					return
				}

				// Forward headers
				req.Header = r.Header

				// Forward the request
				resp, err := client.Do(req)
				if err != nil {
					// Internal error
					log.Printf("Problem posting to relay %q backend %q: %v", h.Name(), b.name, err)

					// So empty response
					responses <- &http.Response{}
				} else {
					if resp.StatusCode/100 == 5 {
						// HTTP error
						log.Printf("5xx response for relay %q backend %q: %v", h.Name(), b.name, resp.StatusCode)
					}

					// Get response
					responses <- resp
				}
			}()
		}

		// Wait for requests
		go func() {
			wg.Wait()
			close(responses)
		}()

		var errResponse *responseData
		for resp := range responses {
			switch resp.StatusCode / 100 {
			case 2:
				w.WriteHeader(http.StatusNoContent)
				return

			case 4:
				// User error
				resp.Write(w)
				return

			default:
				// Hold on to one of the responses to return back to the client
				errResponse = nil
			}
		}

		// No successful writes
		if errResponse == nil {
			// Failed to make any valid request...
			jsonResponse(w, response{http.StatusServiceUnavailable, "unable to forward query"})
			return
		}

	} else { // Bad method
		w.Header().Set("Allow", http.MethodPost)
		jsonResponse(w, response{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)})
		return
	}
}

func (h *HTTP) handleStandard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
		} else {
			jsonResponse(w, response{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)})
			return
		}
	}

	queryParams := r.URL.Query()
	bodyBuf := getBuf()
	_, _ = bodyBuf.ReadFrom(r.Body)

	precision := queryParams.Get("precision")
	points, err := models.ParsePointsWithPrecision(bodyBuf.Bytes(), h.start, precision)
	if err != nil {
		putBuf(bodyBuf)
		jsonResponse(w, response{http.StatusBadRequest, "unable to parse points"})
		return
	}

	outBuf := getBuf()
	for _, p := range points {
		// Those two functions never return any errors, let's just ignore the return value
		_, _ = outBuf.WriteString(p.PrecisionString(precision))
		_ = outBuf.WriteByte('\n')
	}

	// done with the input points
	putBuf(bodyBuf)

	// normalize query string
	query := queryParams.Encode()

	outBytes := outBuf.Bytes()

	// check for authorization performed via the header
	authHeader := r.Header.Get("Authorization")

	var wg sync.WaitGroup
	wg.Add(len(h.backends))

	var responses = make(chan *responseData, len(h.backends))

	for _, b := range h.backends {
		b := b
		if b.inputType != config.TypeInfluxdb {
			wg.Done()
			continue
		}

		go func() {
			defer wg.Done()
			resp, err := b.post(outBytes, query, authHeader)
			if err != nil {
				log.Printf("Problem posting to relay %q backend %q: %v", h.Name(), b.name, err)

				responses <- &responseData{}
			} else {
				if resp.StatusCode/100 == 5 {
					log.Printf("5xx response for relay %q backend %q: %v", h.Name(), b.name, resp.StatusCode)
				}
				responses <- resp
			}
		}()
	}

	go func() {
		wg.Wait()
		close(responses)
		putBuf(outBuf)
	}()

	var errResponse *responseData

	for resp := range responses {
		// Status accepted means buffering,
		// we can handle it early
		if resp.StatusCode == http.StatusAccepted {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		switch resp.StatusCode / 100 {
		case 2:
			w.WriteHeader(http.StatusNoContent)
			return

		case 4:
			// User error
			resp.Write(w)
			return

		default:
			// Hold on to one of the responses to return back to the client
			errResponse = nil
		}
	}

	// No successful writes
	if errResponse == nil {
		// Failed to make any valid request...
		jsonResponse(w, response{http.StatusServiceUnavailable, "unable to write points"})
		return
	}
}

func (h *HTTP) handleProm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
		} else {
			jsonResponse(w, response{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)})
			return
		}
	}

	authHeader := r.Header.Get("Authorization")

	bodyBuf := getBuf()
	_, _ = bodyBuf.ReadFrom(r.Body)

	outBytes := bodyBuf.Bytes()

	var wg sync.WaitGroup
	wg.Add(len(h.backends))

	var responses = make(chan *responseData, len(h.backends))

	for _, b := range h.backends {
		b := b
		if b.inputType != config.TypePrometheus {
			wg.Done()
			continue
		}

		go func() {
			defer wg.Done()
			resp, err := b.post(outBytes, r.URL.RawQuery, authHeader)
			if err != nil {
				log.Printf("Problem posting to relay %q backend %q: %v", h.Name(), b.name, err)

				responses <- &responseData{}
			} else {
				if resp.StatusCode/100 == 5 {
					log.Printf("5xx response for relay %q backend %q: %v", h.Name(), b.name, resp.StatusCode)
				}

				responses <- resp
			}
		}()
	}

	go func() {
		wg.Wait()
		close(responses)
		putBuf(bodyBuf)
	}()

	var errResponse *responseData

	for resp := range responses {
		switch resp.StatusCode / 100 {
		case 2:
			w.WriteHeader(http.StatusNoContent)
			return

		case 4:
			// User error
			resp.Write(w)
			return

		default:
			// Hold on to one of the responses to return back to the client
			errResponse = nil
		}
	}

	// No successful writes
	if errResponse == nil {
		// Failed to make any valid request...
		jsonResponse(w, response{http.StatusServiceUnavailable, "unable to write points"})
		return
	}
}

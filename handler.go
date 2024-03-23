package healthcheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

const (
	// LivenessHandlerPath path to process liveness probe.
	LivenessHandlerPath = "/live"
	// ReadinessHandlerPath path to process readiness probe.
	ReadinessHandlerPath = "/ready"

	successCheckerResultString = "OK"
)

// Handler is a wrapper over http.Handler,
// allowing you to add liveness and readiness checks
type Handler interface {
	// Handler is http.Handler, so it can be exposed directly and processed
	// /live and /ready endpoints.
	http.Handler

	// AddLivenessCheck adds a check indicating that this instance
	// of the application should be destroyed or restarted. A failed liveness check
	// indicates that this instance is not running.
	// Each liveness check is also included as a readiness check.
	AddLivenessCheck(name string, check Check)

	// AddReadinessCheck adds a check indicating that this
	// application instance is currently unable to serve requests due to an external
	// dependency or some kind of temporary failure. If the readiness check fails, this instance
	// should no longer receive requests, but it should not be restarted or destroyed.
	AddReadinessCheck(name string, check Check)

	// LiveEndpoint is an HTTP handler for the /live endpoint only, which
	// is useful if you need to add it to your own HTTP handler tree.
	LiveEndpoint(http.ResponseWriter, *http.Request)

	//ReadyEndpoint is an HTTP handler for the /ready endpoint only, which
	// is useful if you need to add it to your own HTTP handler tree.
	ReadyEndpoint(http.ResponseWriter, *http.Request)

	// AddCheckErrorHandler adds a callback to process a failed check (in order to log errors, etc.).
	AddCheckErrorHandler(handler ErrorHandler)
}

// Check signature of check proccess function
type Check func() error

// ErrorHandler error handler's signature for failed checks.
type ErrorHandler func(name string, err error)

// NewHandler creates a new basic Handler
func NewHandler() Handler {
	h := &basicHandler{
		livenessChecks:  make(map[string]Check),
		readinessChecks: make(map[string]Check),
	}
	h.Handle("/live", http.HandlerFunc(h.LiveEndpoint))
	h.Handle("/ready", http.HandlerFunc(h.ReadyEndpoint))
	return h
}

// basicHandler implementation of Handler.
type basicHandler struct {
	http.ServeMux
	checksMutex     sync.RWMutex
	livenessChecks  map[string]Check
	readinessChecks map[string]Check
	errorHandler    ErrorHandler
}

func (s *basicHandler) LiveEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.livenessChecks)
}

func (s *basicHandler) ReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.readinessChecks, s.livenessChecks)
}

func (s *basicHandler) AddLivenessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.livenessChecks[name] = check
}

func (s *basicHandler) AddReadinessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.readinessChecks[name] = check
}

func (s *basicHandler) AddCheckErrorHandler(handler ErrorHandler) {
	s.errorHandler = handler
}

type result struct {
	name   string
	result string
}

func (s *basicHandler) collectChecks(checks map[string]Check, resultsOut map[string]string) (status int) {
	s.checksMutex.RLock()
	defer s.checksMutex.RUnlock()

	status = http.StatusOK

	if len(checks) == 0 {
		return
	}

	var (
		wg      = sync.WaitGroup{}
		results = make(chan result)
	)

	for name, check := range checks {
		wg.Add(1)

		go func(name string, check Check) {
			defer func() {
				wg.Done()

				// check panic error
				if r := recover(); r != nil {
					results <- result{
						name:   name,
						result: fmt.Sprintf("checker panic recovered: %v", r),
					}

					if s.errorHandler != nil {
						s.errorHandler(name, fmt.Errorf("checker panic recovered: %v", r))
					}
				}
			}()

			var val = successCheckerResultString
			if err := check(); err != nil {
				val = err.Error()

				if s.errorHandler != nil {
					s.errorHandler(name, err)
				}
			}

			results <- result{
				name:   name,
				result: val,
			}
		}(name, check)
	}

	// wait for all checks to be made
	// then close the results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		resultsOut[res.name] = res.result

		if res.result != successCheckerResultString {
			status = http.StatusServiceUnavailable
		}
	}

	return status
}

func (s *basicHandler) handle(w http.ResponseWriter, r *http.Request, checks ...map[string]Check) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checkResults := make(map[string]string)
	status := http.StatusOK
	for _, m := range checks {
		if s := s.collectChecks(m, checkResults); s != http.StatusOK {
			status = s
		}
	}

	// Set response code and content header
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	w.WriteHeader(status)

	// If not ?full=1, we return an empty body. Kubernetes only cares about
	// HTTP status codes, so we won't waste bytes on the full request body.
	if r.URL.Query().Get("full") != "1" {
		_, _ = w.Write([]byte("{}\n"))
		return
	}

	// Write the JSON body, ignoring any encoding errors (which
	// are actually not possible because we encode map[string]string).
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	_ = encoder.Encode(checkResults)
}

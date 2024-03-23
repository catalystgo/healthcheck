package healthcheck

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/catalystgo/healthcheck/mock"
	"github.com/golang/mock/gomock"
)

type errorHandler interface { // nolint  // used for code generation
	Handle(string, error)
}

func TestHandler(t *testing.T) {
	var (
		readyCheck = "test-readiness-check"
		readyErr   = errors.New("failed readiness check")
		liveCheck  = "test-liveness-check"
		liveErr    = errors.New("failed liveness check")
	)

	tests := []struct {
		name       string
		method     string
		path       string
		live       bool
		ready      bool
		expect     int
		expectBody string
		setupMock  func(mock *mock.MockErrorHanlder)
	}{
		{
			name:   "GET /foo should generate a 404",
			method: "POST",
			path:   "/foo",
			live:   true,
			ready:  true,
			expect: http.StatusNotFound,
		},
		{
			name:   "POST /live should generate a 405 Method Not Allowed",
			method: "POST",
			path:   "/live",
			live:   true,
			ready:  true,
			expect: http.StatusMethodNotAllowed,
		},
		{
			name:   "POST /ready should generate a 405 Method Not Allowed",
			method: "POST",
			path:   "/ready",
			live:   true,
			ready:  true,
			expect: http.StatusMethodNotAllowed,
		},
		{
			name:       "with no checks, /live should succeed",
			method:     "GET",
			path:       "/live",
			live:       true,
			ready:      true,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with no checks, /ready should succeed",
			method:     "GET",
			path:       "/ready",
			live:       true,
			ready:      true,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with a failing readiness check, /live should still succeed",
			method:     "GET",
			path:       "/live?full=1",
			live:       true,
			ready:      false,
			expect:     http.StatusOK,
			expectBody: "{}\n",
		},
		{
			name:       "with a failing readiness check, /ready should fail",
			method:     "GET",
			path:       "/ready?full=1",
			live:       true,
			ready:      false,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-readiness-check\": \"failed readiness check\"\n}\n",
			setupMock: func(mock *mock.MockErrorHanlder) {
				mock.EXPECT().Handle(readyCheck, readyErr)
			},
		},
		{
			name:       "with a failing liveness check, /live should fail",
			method:     "GET",
			path:       "/live?full=1",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-liveness-check\": \"failed liveness check\"\n}\n",
			setupMock: func(mock *mock.MockErrorHanlder) {
				mock.EXPECT().Handle(liveCheck, liveErr)
			},
		},
		{
			name:       "with a failing liveness check, /ready should fail",
			method:     "GET",
			path:       "/ready?full=1",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{\n    \"test-liveness-check\": \"failed liveness check\"\n}\n",
			setupMock: func(mock *mock.MockErrorHanlder) {
				mock.EXPECT().Handle(liveCheck, liveErr)
			},
		},
		{
			name:       "with a failing liveness check, /ready without full=1 should fail with an empty body",
			method:     "GET",
			path:       "/ready",
			live:       false,
			ready:      true,
			expect:     http.StatusServiceUnavailable,
			expectBody: "{}\n",
			setupMock: func(mock *mock.MockErrorHanlder) {
				mock.EXPECT().Handle(liveCheck, liveErr)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			errHandler := mock.NewMockErrorHanlder(ctrl)

			h := NewHandler()
			h.AddCheckErrorHandler(errHandler.Handle)

			if !tt.live {
				h.AddLivenessCheck(liveCheck, func() error { return liveErr })

				if tt.setupMock != nil {
					tt.setupMock(errHandler)
				}
			}

			if !tt.ready {
				h.AddReadinessCheck(readyCheck, func() error { return readyErr })

				if tt.setupMock != nil {
					tt.setupMock(errHandler)
				}
			}

			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Errorf("Received unexpected error:\n%+v", err)
			}

			reqStr := tt.method + " " + tt.path
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.expect {
				t.Errorf("Wrong code for %q\n"+
					"expected: %v\n"+
					"actual  : %v", reqStr, tt.expect, rr.Code)
			}

			if tt.expectBody != "" {
				if rr.Body.String() != tt.expectBody {
					t.Errorf("Wrong body for %q\n"+
						"expected: %v"+
						"actual  : %v", reqStr, tt.expectBody, rr.Body.String())
				}
			}
		})
	}
}

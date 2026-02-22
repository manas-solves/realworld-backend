package main

import (
	"net/http"
	"testing"
)

type healthCheckResponse struct {
	Status     string            `json:"status"`
	SystemInfo map[string]string `json:"system_info"`
}

func TestHealthcheckHandler(t *testing.T) {
	t.Parallel()
	want := healthCheckResponse{
		Status: "available",
		SystemInfo: map[string]string{
			"environment": "development",
			"version":     version,
		},
	}

	requestUrlPath := "/healthcheck"

	validResponseTC := handlerTestcase{
		name:                   "Valid response",
		requestMethodType:      http.MethodGet,
		requestUrlPath:         requestUrlPath,
		wantResponseStatusCode: http.StatusOK,
		wantResponse:           want,
	}

	methodNotAllowedTC := handlerTestcase{
		name:                   "Post method not allowed",
		requestMethodType:      http.MethodPost,
		requestUrlPath:         requestUrlPath,
		wantResponseStatusCode: http.StatusMethodNotAllowed,
	}

	invalidUrlPathTC := handlerTestcase{
		name:                   "Invalid URL path",
		requestMethodType:      http.MethodGet,
		requestUrlPath:         "/invalidpath",
		wantResponseStatusCode: http.StatusNotFound,
		wantResponse: errorResponse{
			Errors: []string{"the requested resource could not be found"},
		},
	}

	ts := newTestServer(t)
	testHandler(t, ts, validResponseTC, methodNotAllowedTC, invalidUrlPathTC)
}

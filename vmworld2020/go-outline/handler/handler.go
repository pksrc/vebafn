package function

import (
	"net/http"

	handler "github.com/openfaas/templates-sdk/go-http"
)

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {
	// Parse the event

	// Read the config

	// Implement business logic

	// Handle function response
	return handler.Response{
		Body:       req.Body,
		StatusCode: http.StatusOK,
	}, nil
}

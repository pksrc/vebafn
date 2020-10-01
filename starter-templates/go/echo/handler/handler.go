package function

import (
	"net/http"

	handler "github.com/openfaas/templates-sdk/go-http"
)

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {
	return handler.Response{
		Body:       req.Body,
		StatusCode: http.StatusOK,
	}, nil
}

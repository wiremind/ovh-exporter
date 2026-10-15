package network

import (
	"errors"
	"net/http"

	"github.com/ovh/go-ovh/ovh"
)

// isOVHNotFound reports whether err is an OVH API 404.
func isOVHNotFound(err error) bool {
	var apiError *ovh.APIError

	return errors.As(err, &apiError) && apiError.Code == http.StatusNotFound
}

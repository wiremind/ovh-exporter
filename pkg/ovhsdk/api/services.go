package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/go-ovh/ovh"
)

// Define Options with pointer fields for nullable values.
type Options struct {
	ResourceName *string
	OrderBy      *string
	Routes       *string
	Sort         *string
}

// encodeQueryParam encodes a query parameter value.
func encodeQueryParam(value string) string {
	return url.QueryEscape(value)
}

// GetServices returns the list of available services.
func GetServices(client *ovh.Client, opts *Options) ([]int, error) {
	var service []int

	// Build query parameters if fields are set
	var queryParams []string
	if opts.ResourceName != nil {
		queryParams = append(queryParams, fmt.Sprintf("resourceName=%s", encodeQueryParam(*opts.ResourceName)))
	}
	if opts.OrderBy != nil {
		queryParams = append(queryParams, fmt.Sprintf("orderBy=%s", encodeQueryParam(*opts.OrderBy)))
	}
	if opts.Routes != nil {
		queryParams = append(queryParams, fmt.Sprintf("routes=%s", encodeQueryParam(*opts.Routes)))
	}
	if opts.Sort != nil {
		queryParams = append(queryParams, fmt.Sprintf("sort=%s", encodeQueryParam(*opts.Sort)))
	}

	// Join query parameters and build the query string
	queryString := "?" + strings.Join(queryParams, "&")

	endpoint := "/services" + queryString

	// Fetch services from the API
	err := client.Get(endpoint, &service)
	if err != nil {
		return nil, fmt.Errorf("failed to get services from OVH API: %w", err)
	}

	return service, nil
}

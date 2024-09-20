package cmd

import "runtime"

const (
	defaultVersion = "v0.0.0+unknown"
)

var (
	// Package is filled at linking time
	Package = "github.com/wiremind/ovh-exporter"

	// Version holds the complete version number. Filled in at linking time.
	Version = defaultVersion

	// Revision is filled with the VCS (e.g. git) revision being used to build
	// the program at linking time.
	Revision = ""

	// GoVersion is Go tree's version.
	GoVersion = runtime.Version()
)

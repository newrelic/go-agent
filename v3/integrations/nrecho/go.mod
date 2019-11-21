module github.com/newrelic/go-agent/v3/integrations/nrecho

go 1.13

require (
	// v3.1.0 is the earliest v3 version of Echo that works with modules due
	// to the github.com/rsc/letsencrypt import of v3.0.0.
	github.com/labstack/echo v3.1.0+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/newrelic/go-agent/v3 v3.0.0
	golang.org/x/crypto v0.0.0-20191119213627-4f8c1d86b1ba // indirect
)

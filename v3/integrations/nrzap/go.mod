module github.com/newrelic/go-agent/v3/integrations/nrzap

// As of Jun 2022, zap has 1.18 in their go.mod file:
// https://github.com/uber-go/zap/blob/master/go.mod
go 1.18
require (
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/google/renameio v0.1.0 // indirect
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/newrelic/go-agent/v3 v3.16.1
	github.com/rogpeppe/go-internal v1.3.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/tools v0.0.0-20190618225709-2cfd321de3ee // indirect
	// v1.21.0 is the earliest version of zap using modules.
	go.uber.org/zap v1.21.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/net v0.0.0-20220615171555-694bf12d69de // indirect
	golang.org/x/sys v0.0.0-20220615213510-4f61da869c0c // indirect
	golang.org/x/tools v0.1.11 // indirect
	google.golang.org/genproto v0.0.0-20220616135557-88e70c0c3a90 // indirect
	honnef.co/go/tools v0.3.2 // indirect
)

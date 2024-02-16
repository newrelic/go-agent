By default, building the server application here (e.g., via `go build` or `go run main.go`) will create a simple web service that is instrumented by the Go Agent. This
has a number of HTTP endpoints which demonstrate different capabilities:
   `/`
   `/add_attribute`
   `/add_span_attribute`
   `/async`
   `/background_log`
   `/background`
   `/browser`
   `/custom_event`
   `/custommetric`
   `/external`
   `/ignore`
   `/log`
   `/message`
   `/mysql`
   `/notice_error_with_attributes`
   `/notice_error`
   `/notice_expected_error`
   `/roundtripper`
   `/segments`
   `/set_name`
   `/version`
All of these are served from TCP port 8000 on the local host.

However, if you build the application with `go build -tags control`, you'll get a "control" version which does not use the Go Agent, to compare against if you are testing to see how an app performs with and without the agent.

If you build with `go build -tags profiling`, you will get a version which generates CPU and memory profiling data. You can get both (a profiling, non-agent version) by including both tags as `go build -tags profiling,control`.

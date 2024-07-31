# If it is more convenient for you to run an instrumented test server in a Docker
# container, you can use this Dockerfile to build an image for that purpose.
#
# To build this image, have this Dockerfile in the current directory and run:
#    docker build -t go-agent-test .
#
# To run a test, run the following:
#    docker run -e NEW_RELIC_LICENSE_KEY="YOUR_KEY_HERE" -p 127.0.0.1:8000:8000 go-agent-test
#    then drive traffic to it on localhost port 8000
#
#    This running application will write debugging logs showing all interaction
#    with the collector on its standard output.
#
#    The following HTTP endpoints can be accessed on port 8000 to invoke different
#    instrumented server features:
#	/
#	/add_attribute
#	/add_span_attribute
#	/async
#	/background
#	/background_log
#	/browser
#	/custom_event
#	/custommetric
#	/external
#	/ignore
#	/log
#	/message
#	/mysql
#	/notice_error
#	/notice_error_with_attributes
#	/notice_expected_error
#	/roundtripper
#	/segments
#	/set_name
#	/version
#
FROM golang:1.22
MAINTAINER Steve Willoughby <swilloughby@newrelic.com>
WORKDIR /go
RUN git clone https://github.com/newrelic/go-agent
WORKDIR /go/go-agent/v3
RUN go mod tidy
WORKDIR /go/go-agent/v3/examples/server
RUN go mod tidy
RUN go build
EXPOSE 8000
CMD ["/go/go-agent/v3/examples/server/server"]
#
# END
#

package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/newrelic/go-sdk/api"
	"github.com/newrelic/go-sdk/log"
	"github.com/newrelic/go-sdk/version"
)

const (
	procotolVersion = "14"
	userAgent       = "NewRelic-Go-SDK/" + version.Version

	// Methods used in collector communication.
	cmdRedirect     = "get_redirect_host"
	cmdConnect      = "connect"
	cmdMetrics      = "metric_data"
	cmdCustomEvents = "custom_event_data"
	cmdTxnEvents    = "analytic_event_data"
	cmdErrorEvents  = "error_event_data"
	cmdErrorData    = "error_data"
)

var (
	// ErrPayloadTooLarge is created in response to receiving a 413 response
	// code.
	ErrPayloadTooLarge = errors.New("payload too large")
	// ErrUnsupportedMedia is created in response to receiving a 415
	// response code.
	ErrUnsupportedMedia = errors.New("unsupported media")
)

type Cmd struct {
	Name      string
	UseTLS    bool
	Collector string
	License   string
	RunID     string
	Data      []byte
}

func (cmd *Cmd) url() string {
	var u url.URL

	u.Host = cmd.Collector
	u.Path = "agent_listener/invoke_raw_method"

	if cmd.UseTLS {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}

	query := url.Values{}
	query.Set("marshal_format", "json")
	query.Set("protocol_version", procotolVersion)
	query.Set("method", cmd.Name)
	query.Set("license_key", cmd.License)

	if len(cmd.RunID) > 0 {
		query.Set("run_id", cmd.RunID)
	}

	u.RawQuery = query.Encode()
	return u.String()
}

type unexpectedStatusCodeErr struct {
	code int
}

func (e unexpectedStatusCodeErr) Error() string {
	return fmt.Sprintf("unexpected HTTP status code: %d", e.code)
}

func collectorRequestInternal(url string, data []byte, client *http.Client) ([]byte, error) {
	deflated, err := compress(data)
	if nil != err {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(deflated))
	if nil != err {
		return nil, err
	}

	req.Header.Add("Accept-Encoding", "identity, deflate")
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Content-Encoding", "deflate")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if 413 == resp.StatusCode {
		return nil, ErrPayloadTooLarge
	}

	if 415 == resp.StatusCode {
		return nil, ErrUnsupportedMedia
	}

	// If the response code is not 200, then the collector may not return
	// valid JSON.
	if 200 != resp.StatusCode {
		return nil, unexpectedStatusCodeErr{code: resp.StatusCode}
	}

	b, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return nil, err
	}
	return parseResponse(b)
}

func collectorRequest(cmd Cmd, client *http.Client) ([]byte, error) {
	url := cmd.url()

	cn := log.Context{
		"command": cmd.Name,
		"url":     url,
	}

	log.Debug("rpm request", cn, log.Context{
		"payload": JSONString(cmd.Data),
	})

	resp, err := collectorRequestInternal(url, cmd.Data, client)
	if err != nil {
		log.Debug("rpm failure", cn, log.Context{"error": err.Error()})
	}

	log.Debug("rpm response", cn, log.Context{
		"response": JSONString(resp),
	})

	return resp, err
}

type rpmException struct {
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
}

func (e *rpmException) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorType, e.Message)
}

func hasType(e error, expected string) bool {
	rpmErr, ok := e.(*rpmException)
	if !ok {
		return false
	}
	return rpmErr.ErrorType == expected

}

const (
	forceRestartType   = "NewRelic::Agent::ForceRestartException"
	disconnectType     = "NewRelic::Agent::ForceDisconnectException"
	licenseInvalidType = "NewRelic::Agent::LicenseException"
	runtimeType        = "RuntimeError"
)

func isRestartException(e error) bool { return hasType(e, forceRestartType) }
func isLicenseException(e error) bool { return hasType(e, licenseInvalidType) }
func isRuntime(e error) bool          { return hasType(e, runtimeType) }
func isDisconnect(e error) bool       { return hasType(e, disconnectType) }

func parseResponse(b []byte) ([]byte, error) {
	var r struct {
		ReturnValue json.RawMessage `json:"return_value"`
		Exception   *rpmException   `json:"exception"`
	}

	err := json.Unmarshal(b, &r)
	if nil != err {
		return nil, err
	}

	if nil != r.Exception {
		return nil, r.Exception
	}

	return r.ReturnValue, nil
}

func processConnectMessages(reply []byte) {
	var msgs struct {
		Messages []struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		} `json:"messages"`
	}

	err := json.Unmarshal(reply, &msgs)
	if nil != err {
		return
	}

	for _, msg := range msgs.Messages {
		event := "collector message"
		cn := log.Context{"msg": msg.Message}

		switch strings.ToLower(msg.Level) {
		case "error":
			log.Error(event, cn)
		case "warn":
			log.Warn(event, cn)
		case "info":
			log.Info(event, cn)
		case "debug", "verbose":
			log.Debug(event, cn)
		}
	}
}

func connectAttempt(cfg *api.Config, client *http.Client) (string, *ConnectReply, error) {
	js, err := configConnectJSON(cfg)
	if nil != err {
		return "", nil, err
	}

	call := Cmd{
		Name:      cmdRedirect,
		UseTLS:    cfg.UseTLS,
		Collector: redirectHost,
		License:   cfg.License,
		Data:      []byte("[]"),
	}

	out, err := collectorRequest(call, client)
	if nil != err {
		// err is intentionally unmodified:  We do not want to change
		// the type of these collector errors.
		return "", nil, err
	}

	var host string
	err = json.Unmarshal(out, &host)
	if nil != err {
		return "", nil, fmt.Errorf("unable to parse redirect reply: %v", err)
	}

	call.Collector = host
	call.Data = js
	call.Name = cmdConnect

	rawReply, err := collectorRequest(call, client)
	if nil != err {
		// err is intentionally unmodified:  We do not want to change
		// the type of these collector errors.
		return "", nil, err
	}

	processConnectMessages(rawReply)

	reply := ConnectReplyDefaults()
	err = json.Unmarshal(rawReply, reply)
	if nil != err {
		return "", nil, fmt.Errorf("unable to parse connect reply: %v", err)
	}
	// Note:  This should never happen.  It would mean the collector
	// response is malformed.  This exists merely as extra defensiveness.
	if "" == reply.RunID {
		return "", nil, errors.New("connect reply missing agent run id")
	}

	return host, reply, nil
}

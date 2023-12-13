package newrelic

import (
	"net/http"
)

// secureAgent is a global interface point for the nrsecureagent's hooks into the go agent.
// The default value for this is a noOpSecurityAgent value, which has null definitions for
// the methods. The Go compiler is expected to optimize away all the securityAgent method
// calls in this case, effectively removing the hooks from the running agent.
//
// If the nrsecureagent integration was initialized, it will register a real securityAgent
// value in the securityAgent varialble instead, thus "activating" the hooks.
var secureAgent securityAgent = noOpSecurityAgent{}

// GetSecurityAgentInterface returns the securityAgent value
// which provides the working interface to the installed
// security agent (or to a no-op interface if none were
// installed).
//
// Packages which need to make calls to secureAgent's methods
// may obtain the secureAgent value by calling this function.
// This avoids exposing the variable itself so it's not
// writable externally and also sets up for the future if this
// ends up not being a global variable later.
func GetSecurityAgentInterface() securityAgent {
	return secureAgent
}

type securityAgent interface {
	RefreshState(map[string]string) bool
	DeactivateSecurity()
	SendEvent(string, ...any) any
	IsSecurityActive() bool
	DistributedTraceHeaders(hdrs *http.Request, secureAgentevent any)
	SendExitEvent(any, error)
	RequestBodyReadLimit() int
}

func (app *Application) RegisterSecurityAgent(s securityAgent) {
	if app != nil && app.app != nil && s != nil {
		secureAgent = s
		if app.app.run != nil {
			secureAgent.RefreshState(getLinkedMetaData(app.app))
		}
	}
}

func (app *Application) UpdateSecurityConfig(s interface{}) {
	if app == nil || app.app == nil {
		return
	}
	app.app.config.Config.Security = s
}

func getLinkedMetaData(app *app) map[string]string {
	runningAppData := make(map[string]string)
	if app != nil && app.run != nil {
		runningAppData["hostname"] = app.run.Config.hostname
		runningAppData["entityName"] = app.run.firstAppName
		if app.run != nil {
			runningAppData["entityGUID"] = app.run.Reply.EntityGUID
			runningAppData["agentRunId"] = app.run.Reply.RunID.String()
			runningAppData["accountId"] = app.run.Reply.AccountID
		}
	}
	return runningAppData
}

// noOpSecurityAgent satisfies the secureAgent interface but is a null implementation
// that will largely be optimized away at compile time.
type noOpSecurityAgent struct {
}

func (t noOpSecurityAgent) RefreshState(connectionData map[string]string) bool {
	return false
}

func (t noOpSecurityAgent) DeactivateSecurity() {
}

func (t noOpSecurityAgent) SendEvent(caseType string, data ...any) any {
	return nil
}

func (t noOpSecurityAgent) IsSecurityActive() bool {
	return false
}

func (t noOpSecurityAgent) DistributedTraceHeaders(hdrs *http.Request, secureAgentevent any) {
}

func (t noOpSecurityAgent) SendExitEvent(secureAgentevent any, err error) {
}
func (t noOpSecurityAgent) RequestBodyReadLimit() int {
	return 300 * 1000
}

// IsSecurityAgentPresent returns true if there's an actual security agent hooked in to the
// Go APM agent, whether or not it's enabled or operating in any particular mode. It returns
// false only if the hook-in interface for those functions is a No-Op will null functionality.
func IsSecurityAgentPresent() bool {
	_, isNoOp := secureAgent.(noOpSecurityAgent)
	return !isNoOp
}

type BodyBuffer struct {
	buf             []byte
	isDataTruncated bool
}

func (b *BodyBuffer) Write(p []byte) (int, error) {
	if l := len(b.buf); len(p) <= secureAgent.RequestBodyReadLimit()-l {
		b.buf = append(b.buf, p...)
		return len(p), nil
	} else if l := len(b.buf); secureAgent.RequestBodyReadLimit()-l > 1 {
		end := secureAgent.RequestBodyReadLimit() - l
		b.buf = append(b.buf, p[:end-1]...)
		return end, nil
	} else {
		b.isDataTruncated = true
		return 0, nil
	}
}

func (b *BodyBuffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.buf)

}

func (b *BodyBuffer) read() []byte {
	if b == nil {
		return make([]byte, 0)
	}
	return b.buf
}

func (b *BodyBuffer) isBodyTruncated() bool {
	if b == nil {
		return false
	}
	return b.isDataTruncated
}
func (b *BodyBuffer) String() (string, bool) {
	if b == nil {
		return "", false
	}
	return string(b.buf), b.isDataTruncated

}

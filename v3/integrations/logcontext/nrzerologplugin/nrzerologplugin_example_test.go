package nrzerologplugin_test

import (
	"net/http"
	"os"

	"github.com/justinas/alice"
	"github.com/newrelic/go-agent/v3/integrations/logcontext/nrzerologplugin"
	"github.com/rs/zerolog"
)

func ExampleNrZerologPlugin() {
	logger := zerolog.New(os.Stdout)
	logger = logger.Hook(zerolog.HookFunc(nrzerologplugin.Hook))

	myHandler := func(w http.ResponseWriter, r *http.Request) {
		logger := zerolog.Ctx(r.Context())
		logger.Info().Msg("hello world")
		w.Write([]byte("hello world"))
	}

	chain := alice.New(nrzerologplugin.Middleware).Then(http.HandlerFunc(myHandler))

	http.ListenAndServe(":8000", chain)
}

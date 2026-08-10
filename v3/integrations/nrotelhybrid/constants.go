package nrotelhybrid

// OTel semantic convention attribute keys used by this package.
//
// These are hardcoded rather than imported from go.opentelemetry.io/otel/semconv
// because the bridge spec (Tracing-API.md) requires mapping both the current
// stable semantic convention and legacy "pinned" versions side by side, and a
// single semconv/vX.Y.Z package only exposes one version's keys at a time.
const (
	// shared attributes

	// specific attributes
	AttrDBSystemName = "db.system.name" // OTEL DB Client v1.25+ (stable)
	AttrDBSystem     = "db.system"      // OTEL DB Client v1.17 (legacy)

	AttrHTTPStatusCode = "http.status_code" // OTEL HTTP Client 1.17
	AttrHTTPStatusText = "http.status_text" // OTEL HTTP Client 1.17
	AttrHTTPMethod     = "http.method"      // OTEL HTTP Client 1.17
	AttrHTTPURL        = "http.url"         // OTEL HTTP Client 1.17
	AttrServerAddress  = "server.address"   // OTEL HTTP Client 1.17
	AttrServerPort     = "server.port"      // OTEL HTTP Client 1.17

	AttrHTTPResponseStatusCode = "http.response.status_code" // OTEL HTTP Client 1.23
	AttrHTTPResponseStatusText = "http.response.status_text" // OTEL HTTP Client 1.23
	AttrHTTPRequestMethod      = "http.request.method"       // OTEL HTTP Client 1.23
	AttrURLFull                = "url.full"                  // OTEL HTTP Client 1.23
	AttrNetPeerName            = "net.peer.name"             // OTEL HTTP Client 1.23
	AttrNetPeerPort            = "net.peer.port"             // OTEL HTTP Client 1.23

)

// NR segment/transaction attribute keys used by this package.
const (
	NRHTTPStatusCode = "http.statusCode"
	NRHTTPStatusText = "http.statusText"
	NRProcedure      = "procedure"
	NRHTTPUrl        = "http.url"
	NRHost           = "host"
	NRPort           = "port"
)

// OTELToNRAttributeMap maps OTel attribute keys to their NR segment attribute equivalents.
var OTELToNRAttributeMap = map[string]string{
	// OTEL HTTP Client 1.17
	AttrHTTPStatusCode: NRHTTPStatusCode,
	AttrHTTPStatusText: NRHTTPStatusText,
	AttrHTTPMethod:     NRProcedure,
	AttrHTTPURL:        NRHTTPUrl,
	AttrServerAddress:  NRHost,
	AttrServerPort:     NRPort,
	// OTEL HTTP Client 1.23
	AttrHTTPResponseStatusCode: NRHTTPStatusCode,
	AttrHTTPResponseStatusText: NRHTTPStatusText,
	AttrHTTPRequestMethod:      NRProcedure,
	AttrURLFull:                NRHTTPUrl,
	AttrNetPeerName:            NRHost,
	AttrNetPeerPort:            NRPort,
}

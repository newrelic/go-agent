package nrotelhybrid

// OTel semantic convention attribute keys used by this package.
//
// These are hardcoded rather than imported from go.opentelemetry.io/otel/semconv
// because the bridge spec (Tracing-API.md) requires mapping both the current
// stable semantic convention and legacy "pinned" versions side by side, and a
// single semconv/vX.Y.Z package only exposes one version's keys at a time.
const (
	AttrURLFull        = "url.full"        // OTel HTTP Client v1.23+
	AttrDBSystemName   = "db.system.name"  // OTel DB Client v1.25+ (stable)
	AttrDBSystem       = "db.system"       // OTel DB Client v1.17 (legacy)
	AttrHTTPStatusCode = "http.status_code" // OTel HTTP Server/Client v1.20 (legacy)
)

// NR segment/transaction attribute keys used by this package.
const (
	NRHTTPStatusCode = "http.statusCode"
)

// OTELToNRAttributeMap maps OTel attribute keys to their NR segment attribute equivalents.
var OTELToNRAttributeMap = map[string]string{
	AttrHTTPStatusCode: NRHTTPStatusCode,
}
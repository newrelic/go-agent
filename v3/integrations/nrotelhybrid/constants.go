package nrotelhybrid

// OTel semantic convention attribute keys used by this package.
//
// These are hardcoded rather than imported from go.opentelemetry.io/otel/semconv
// because the bridge spec (Tracing-API.md) requires mapping both the current
// stable semantic convention and legacy "pinned" versions side by side, and a
// single semconv/vX.Y.Z package only exposes one version's keys at a time.
const (
	// shared attributes

	AttrServerAddress = "server.address" // HTTP Client v1.17, DB Client v1.25 (DB/Redis/Mongo)
	AttrServerPort    = "server.port"    // HTTP Client v1.17, DB Client v1.25 (DB/Redis/Mongo)
	AttrNetPeerName   = "net.peer.name"  // HTTP Client v1.23, DB Client v1.17 (DB/Redis/Mongo)
	AttrNetPeerPort   = "net.peer.port"  // HTTP Client v1.23, DB Client v1.17 (DB/Redis/Mongo/Dynamo)

	// specific attributes

	AttrHTTPStatusCode = "http.status_code" // OTEL HTTP Client v1.17
	AttrHTTPStatusText = "http.status_text" // OTEL HTTP Client v1.17
	AttrHTTPMethod     = "http.method"      // OTEL HTTP Client v1.17
	AttrHTTPURL        = "http.url"         // OTEL HTTP Client v1.17

	AttrHTTPResponseStatusCode = "http.response.status_code" // OTEL HTTP Client v1.23
	AttrHTTPResponseStatusText = "http.response.status_text" // OTEL HTTP Client v1.23
	AttrHTTPRequestMethod      = "http.request.method"       // OTEL HTTP Client v1.23
	AttrURLFull                = "url.full"                  // OTEL HTTP Client v1.23

	AttrDBSystemName = "db.system.name" // OTEL DB Client v1.25 (DB/Redis/Mongo)
	AttrDBNamespace  = "db.namespace"   // OTEL DB Client v1.25 (DB/Mongo)

	AttrDBSystem = "db.system" // OTEL DB Client v1.17 (DB/Redis/Mongo/Dynamo)
	AttrDBName   = "db.name"   // OTEL DB Client v1.17 (DB/Redis/Mongo/Dynamo)

)

// NR segment/transaction attribute keys used by this package.
const (
	NRHTTPStatusCode = "http.statusCode"
	NRHTTPStatusText = "http.statusText"
	NRProcedure      = "procedure"
	NRHTTPUrl        = "http.url"
	NRHost           = "host"
	NRPort           = "port"

	NRProduct      = "product"
	NRDatabaseName = "database_name"
	NRPortPathOrID = "port_path_or_id"
)

// OTELToNRHTTPAttributeMap maps OTel HTTP client/server attribute keys to their
// NR segment attribute equivalents (OTel HTTP Client v1.17 and v1.23).
var OTELToNRHTTPAttributeMap = map[string]string{
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

// OTELToNRDBAttributeMap maps OTel DB client attribute keys to their NR
// datastore segment attribute equivalents (OTel DB/Redis/Mongo/Dynamo Client
// v1.17 and v1.25).
var OTELToNRDBAttributeMap = map[string]string{
	// OTEL DB Client v1.25 (DB/Redis/Mongo)
	AttrDBSystemName:  NRProduct,
	AttrDBNamespace:   NRDatabaseName, // Not used by Redis
	AttrServerAddress: NRHost,
	AttrServerPort:    NRPortPathOrID,
	// OTEL DB Client v1.17 (DB/Redis/Mongo/Dynamo)
	AttrDBSystem:    NRProduct,
	AttrDBName:      NRDatabaseName,
	AttrNetPeerName: NRHost, // Not used by Dynamo
	AttrNetPeerPort: NRPortPathOrID,
}

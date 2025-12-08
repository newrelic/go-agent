// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package nroci

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/oracle/nosql-go-sdk/nosqldb"
	"github.com/oracle/nosql-go-sdk/nosqldb/auth/iam"
)

func init() {
	//add more here
}

var profileRegex = regexp.MustCompile(`^\[(.*)\]`) // from nosql-go-sdk from OCI
var ociFilePath = "~/.oci/config/"                 // default path for OCI Config file

type OCIClient interface {
	AddReplica(req *nosqldb.AddReplicaRequest) (*nosqldb.TableResult, error)
	Close() error
	Delete(req *nosqldb.DeleteRequest) (*nosqldb.DeleteResult, error)
	DoTableRequest(req *nosqldb.TableRequest) (*nosqldb.TableResult, error)
	DoTableRequestAndWait(req *nosqldb.TableRequest, timeout time.Duration, pollInterval time.Duration) (*nosqldb.TableResult, error)
	DropReplica(req *nosqldb.DropReplicaRequest) (*nosqldb.TableResult, error)

	Get(req *nosqldb.GetRequest) (*nosqldb.GetResult, error)
	GetTable(req *nosqldb.GetTableRequest) (*nosqldb.TableResult, error)

	MultiDelete(req *nosqldb.MultiDeleteRequest) (*nosqldb.MultiDeleteResult, error)
	Put(req *nosqldb.PutRequest) (*nosqldb.PutResult, error)
	Query(req *nosqldb.QueryRequest) (*nosqldb.QueryResult, error)

	WriteMultiple(req *nosqldb.WriteMultipleRequest) (*nosqldb.WriteMultipleResult, error)
}

type OCIProfile struct {
	Profile string
	Info    *OCIProfileInfo
}

type OCIProfileInfo struct {
	TenancyOCID string
}

type ConfigWrapper struct {
	Config        *nosqldb.Config
	CompartmentID string
	Profile       *OCIProfile
}

type ClientWrapper struct {
	Client OCIClient
	Config *ConfigWrapper
}

type ClientRequestWrapper[R any] struct {
	ClientRequest R
}
type ClientResponseWrapper[T any] struct {
	ClientResponse T
}

// uses default file location and DEFAULT profile
func NRConfig(mode string) (*ConfigWrapper, error) {
	profile, err := parseOCIConfig(ociFilePath)
	if err != nil {
		return nil, err
	}
	return &ConfigWrapper{
		Config: &nosqldb.Config{
			Mode: mode,
		},
		Profile: profile,
	}, nil
}

// uses explicit file location and optional profile
func NRConfigFromFile(mode string, configFile string, ociProfile ...string) (*ConfigWrapper, error) {
	profile, err := parseOCIConfig(configFile, ociProfile...)
	if err != nil {
		return nil, err
	}
	return &ConfigWrapper{
		Config: &nosqldb.Config{
			Mode: mode,
		},
		Profile: profile,
	}, nil
}

func NRCreateClient(cfg *ConfigWrapper) (*ClientWrapper, error) {
	client, err := nosqldb.NewClient(*cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("error creating OCI Client: %s", err.Error())
	}
	return &ClientWrapper{
		Client: client,
		Config: cfg,
	}, nil
}

func parseOCIConfig(configFilePath string, ociProfile ...string) (*OCIProfile, error) {
	data, err := openConfigFile(configFilePath)
	if err != nil {
		return nil, err
	}
	profile := "DEFAULT"
	if len(ociProfile) > 0 {
		profile = ociProfile[0]
	}
	info, err := parseOCIConfigFile(data, profile)
	if err != nil {
		return nil, err
	}
	return &OCIProfile{
		Profile: profile,
		Info:    info,
	}, nil
}

func openConfigFile(fp string) ([]byte, error) {
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func parseOCIConfigFile(data []byte, profile string) (*OCIProfileInfo, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty file error")
	}

	content := string(data)
	splitContent := strings.Split(content, "\n")

	for i, line := range splitContent {
		if match := profileRegex.FindStringSubmatch(line); len(match) > 1 && match[1] == profile {
			start := i + 1
			return parseConfigAtLine(start, splitContent)

		}
	}
	return nil, fmt.Errorf("couldn't parse config file")
}

func parseConfigAtLine(start int, splitContent []string) (info *OCIProfileInfo, err error) {
	info = &OCIProfileInfo{}
	for i := start; i < len(splitContent); i++ {
		line := splitContent[i]
		if profileRegex.MatchString(line) {
			break
		}
		if !strings.Contains(line, "=") {
			continue
		}

		splits := strings.Split(line, "=")
		switch key, value := strings.TrimSpace(splits[0]), strings.TrimSpace(splits[1]); strings.ToLower(key) {
		case "tenancy":
			info.TenancyOCID = value
		}
	}
	return info, nil
}

// extractHostPort extracts host and port from an endpoint URL
func extractHostPort(endpoint string) (host, port string) {
	if endpoint == "" {
		return "", ""
	}

	// Parse the endpoint URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", ""
	}

	host = parsedURL.Hostname()
	port = parsedURL.Port()

	// Set default ports if not specified
	if port == "" {
		switch parsedURL.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}

	return host, port
}

func extractRequestFields(req any) (string, string) {
	var collection, statement string
	switch r := req.(type) {
	case *nosqldb.TableRequest:
		collection = r.TableName
		statement = r.Statement
	case *nosqldb.QueryRequest:
		collection = r.TableName
		statement = r.Statement
	case *nosqldb.PutRequest:
		collection = r.TableName
	case *nosqldb.WriteMultipleRequest:
		collection = r.TableName
	case *nosqldb.DeleteRequest:
		collection = r.TableName
	case *nosqldb.MultiDeleteRequest:
		collection = r.TableName
	case *nosqldb.AddReplicaRequest:
		collection = r.TableName
	default:
		// keep strings empty
	}
	return collection, statement
}

// executeWithDatastoreSegment is a generic helper function that executes a query with a given function from the
// OCI Client.  It takes a type parameter T as any because of the different response types that are used within the
// OCI Client.  This function will take the transaction from the context (if it exists) and create a Datastore Segment.
// It will then call whatever client function has been passed in.
func executeWithDatastoreSegment[T any, R any](
	cw *ClientWrapper,
	ctx context.Context,
	rw *ClientRequestWrapper[R],
	fn func() (T, error),
) (*ClientResponseWrapper[T], error) {

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return nil, fmt.Errorf("error executing DoTableRequest, no transaction")
	}

	// Extract host and port from config endpoint
	host, port := extractHostPort(cw.Config.Config.Endpoint)
	collection, statement := extractRequestFields(rw.ClientRequest)
	sgmt := newrelic.DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            newrelic.DatastoreOracle,
		Collection:         collection,
		ParameterizedQuery: statement,
		DatabaseName:       cw.Config.CompartmentID,
		Host:               host,
		PortPathOrID:       port,
	}

	responseWrapper := ClientResponseWrapper[T]{}
	res, err := fn() // call the client function
	responseWrapper.ClientResponse = res
	if err != nil {
		return &responseWrapper, fmt.Errorf("error making request: %s", err.Error())
	}

	sgmt.End()

	return &responseWrapper, nil
}

// Wrapper for nosqldb.Client.DoTableRequest.  Provide the ClientWrapper and Context as parameters in addition to the nosqldb.TableRequest.
// Returns a ClientResponseWrapper[*nosqldb.TableResult] and error.
func NRDoTableRequest(cw *ClientWrapper, ctx context.Context, req *nosqldb.TableRequest) (*ClientResponseWrapper[*nosqldb.TableResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.TableRequest]{ClientRequest: req}, func() (*nosqldb.TableResult, error) {
		return cw.Client.DoTableRequest(req)
	})
}

// Wrapper for nosqldb.Client.DoTableRequestWait.  Provide the ClientWrapper and Context as parameters in addition to the nosqldb.TableRequest,
// timeout, and pollInterval. Returns a ClientResponseWrapper[*nosqldb.TableResult] and error.
func NRDoTableRequestAndWait(cw *ClientWrapper, ctx context.Context, req *nosqldb.TableRequest, timeout time.Duration, pollInterval time.Duration) (*ClientResponseWrapper[*nosqldb.TableResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.TableRequest]{ClientRequest: req}, func() (*nosqldb.TableResult, error) {
		return cw.Client.DoTableRequestAndWait(req, timeout, pollInterval)
	})
}

// Wrapper for nosqldb.Client.Query. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.QueryRequest.  Returns a
// ClientResponseWrapper[*nosqldb.QueryResult] and error
func NRQuery(cw *ClientWrapper, ctx context.Context, req *nosqldb.QueryRequest) (*ClientResponseWrapper[*nosqldb.QueryResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.QueryRequest]{ClientRequest: req}, func() (*nosqldb.QueryResult, error) {
		return cw.Client.Query(req)
	})
}

// Wrapper for nosqldb.Client.Put. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.PutRequest. Returns a
// ClientResponseWrapper[*nosqldb.PutResult] and error
func NRPut(cw *ClientWrapper, ctx context.Context, req *nosqldb.PutRequest) (*ClientResponseWrapper[*nosqldb.PutResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.PutRequest]{ClientRequest: req}, func() (*nosqldb.PutResult, error) {
		return cw.Client.Put(req)
	})
}

// Wrapper for nosqldb.Client.WriteMultiple. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.WriteMultipleRequest. Returns a
// ClientResponseWrapper[*nosqldb.WriteMultipleResult] and error
func NRWriteMultiple(cw *ClientWrapper, ctx context.Context, req *nosqldb.WriteMultipleRequest) (*ClientResponseWrapper[*nosqldb.WriteMultipleResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.WriteMultipleRequest]{ClientRequest: req}, func() (*nosqldb.WriteMultipleResult, error) {
		return cw.Client.WriteMultiple(req)
	})
}

// Wrapper for nosqldb.Client.Delete. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.DeleteRequest. Returns a
// ClientResponseWrapper[*nosqldb.DeleteResult] and error
func NRDelete(cw *ClientWrapper, ctx context.Context, req *nosqldb.DeleteRequest) (*ClientResponseWrapper[*nosqldb.DeleteResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.DeleteRequest]{ClientRequest: req}, func() (*nosqldb.DeleteResult, error) {
		return cw.Client.Delete(req)
	})
}

// Wrapper for nosqldb.Client.MutliDelete. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.DeleteRequest. Returns a
// ClientResponseWrapper[*nosqldb.DeleteResult] and error
func NRMultiDelete(cw *ClientWrapper, ctx context.Context, req *nosqldb.MultiDeleteRequest) (*ClientResponseWrapper[*nosqldb.MultiDeleteResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.MultiDeleteRequest]{ClientRequest: req}, func() (*nosqldb.MultiDeleteResult, error) {
		return cw.Client.MultiDelete(req)
	})
}

// Wrapper for nosqldb.Client.AddReplica. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.AddReplicaRequest. Returns a
// ClientResponseWrapper[*nosqldb.TableResult] and error
func NRAddReplica(cw *ClientWrapper, ctx context.Context, req *nosqldb.AddReplicaRequest) (*ClientResponseWrapper[*nosqldb.TableResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.AddReplicaRequest]{ClientRequest: req}, func() (*nosqldb.TableResult, error) {
		return cw.Client.AddReplica(req)
	})
}

// Wrapper for nosqldb.Client.DropReplica. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.DropReplicaRequest. Returns a
// ClientResponseWrapper[*nosqldb.TableResult] and error
func NRDropReplica(cw *ClientWrapper, ctx context.Context, req *nosqldb.DropReplicaRequest) (*ClientResponseWrapper[*nosqldb.TableResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.DropReplicaRequest]{ClientRequest: req}, func() (*nosqldb.TableResult, error) {
		return cw.Client.DropReplica(req)
	})
}

func newSignatureProvider(cfgWrapper *ConfigWrapper, fn func() (*iam.SignatureProvider, error), compartmentID ...string) (*iam.SignatureProvider, error) {
	sp, err := fn() // call SignatureProvider function
	if err != nil {
		return nil, err
	}
	cfgWrapper.Config.AuthorizationProvider = sp // set the authorization provider
	tenancyOCID, err := sp.Profile().TenancyOCID()
	if err != nil {
		return nil, err
	}
	if tenancyOCID != cfgWrapper.Profile.Info.TenancyOCID {
		cfgWrapper.Profile.Info.TenancyOCID = tenancyOCID // set to what is found by OCI
	}

	// compartmentID is optional; if empty, the tenancyOCID is used in its place. If specified, it represents a compartment id or name.
	if len(compartmentID) > 0 {
		cfgWrapper.CompartmentID = compartmentID[0]
	} else {
		cfgWrapper.CompartmentID = tenancyOCID
	}
	return sp, nil
}

// Wrapper for iam.NewSignatureProvider. Only a *ConfigWrapper is passed in and will automatically call check OCI Config file default location (~/.oci/config) for
// config options. Sets the value of *ConfigWrapper.CompartmentID and returns an *iam.SignatureProvider.
func NRNewSignatureProvider(cfgWrapper *ConfigWrapper) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewSignatureProvider()
	})
}

// Wrapper for iam.NewSignatureProviderFromFile. Requires configFilePath, ociProfile, privateKeyPassphrase, and compartmentID as parameters.
// Calls newSignatureProvider with iam.NewSignatureProviderFromFile to set the cfgWrapper.CompartmentID and return *iam.SignatureProvider
func NRNewSignatureProviderFromFile(cfgWrapper *ConfigWrapper, configFilePath string, ociProfile string, privateKeyPassphrase string, compartmentID string) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewSignatureProviderFromFile(configFilePath, ociProfile, privateKeyPassphrase, compartmentID)
	}, compartmentID)
}

// Wrapper for iam.NewSignatureProvider. Requires *ConfigWrapper, tenancy, user, region, fingerprint, compartmentID, privateKeyOrFile and privateKeyPassphrase.
// Calls newSignatureProvider with iam.NewRawSignatureProvider to set the cfgWrapper.CompartmentID and return *iam.SignatureProvider
func NRNewRawSignatureProvider(cfgWrapper *ConfigWrapper, tenancy string, user string, region string, fingerprint string, compartmentID string, privateKeyOrFile string, privateKeyPassphrase *string) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewRawSignatureProvider(tenancy, user, region, fingerprint, compartmentID, privateKeyOrFile, privateKeyPassphrase)
	}, compartmentID)
}

// Wrapper for iam.NewSignatureProviderWithResourcePrincipal. Requires *ConfigWrapper and compartmentID.
// Calls newSignatureProvider with iam.NewSignatureProviderWithResourcePrincipal to set the cfgWrapper.CompartmentID and return *iam.SignatureProvider
func NRNewSignatureProviderWithResourcePrincipal(cfgWrapper *ConfigWrapper, compartmentID string) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewSignatureProviderWithResourcePrincipal(compartmentID)
	}, compartmentID)
}

// Wrapper for iam.NRNewSignatureProviderWithInstancePrincipal. Requires *ConfigWrapper and compartmentID.
// Calls newSignatureProvider with iam.NRNewSignatureProviderWithInstancePrincipal to set the cfgWrapper.CompartmentID and return *iam.SignatureProvider
func NRNewSignatureProviderWithInstancePrincipal(cfgWrapper *ConfigWrapper, compartmentID string) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewSignatureProviderWithInstancePrincipal(compartmentID)
	}, compartmentID)
}

// Wrapper for iam.NRNewSignatureProviderWithInstancePrincipalDelegation. Requires *ConfigWrapper and compartmentID and delegationToken.
// Calls newSignatureProvider with iam.NRNewSignatureProviderWithInstancePrincipalDelegation to set the cfgWrapper.CompartmentID and return *iam.SignatureProvider
func NRNewSignatureProviderWithInstancePrincipalDelegation(cfgWrapper *ConfigWrapper, compartmentID string, delegationToken string) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewSignatureProviderWithInstancePrincipalDelegation(compartmentID, delegationToken)
	}, compartmentID)
}

// Wrapper for iam.NRNewSignatureProviderWithInstancePrincipalDelegationFromFile. Requires *ConfigWrapper and compartmentID and delegationTokenFile.
// Calls newSignatureProvider with iam.NRNewSignatureProviderWithInstancePrincipalDelegationFromFile to set the cfgWrapper.CompartmentID and return *iam.SignatureProvider
func NRNewSignatureProviderWithInstancePrincipalDelegationFromFile(cfgWrapper *ConfigWrapper, compartmentID string, delegationTokenFile string) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewSignatureProviderWithInstancePrincipalDelegationFromFile(compartmentID, delegationTokenFile)
	}, compartmentID)
}

// Wrapper for iam.NRNewSessionTokenSignatureProvider. Requires *ConfigWrapper.
// Calls newSignatureProvider with iam.NewSessionTokenSignatureProvider to set the cfgWrapper.CompartmentID return the *iam.SignatureProvider
func NRNewSessionTokenSignatureProvider(cfgWrapper *ConfigWrapper) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewSessionTokenSignatureProvider()
	})
}

// Wrapper for iam.NRNewSessionTokenSignatureProvider. Requires *ConfigWrapper, configFilePath, ociProfile and privateKeyPassphrase.
// Calls newSignatureProvider with iam.NewSessionTokenSignatureProvider to set the cfgWrapper.CompartmentID return the *iam.SignatureProvider
func NRNewSessionTokenSignatureProviderFromFile(cfgWrapper *ConfigWrapper, configFilePath string, ociProfile string, privateKeyPassphrase string) (*iam.SignatureProvider, error) {
	return newSignatureProvider(cfgWrapper, func() (*iam.SignatureProvider, error) {
		return iam.NewSessionTokenSignatureProviderFromFile(configFilePath, ociProfile, privateKeyPassphrase)
	})
}

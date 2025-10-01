/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

const (
	dgraphScheme     = "dgraph"
	cloudAPIKeyParam = "apikey"      // optional parameter for providing a Dgraph Cloud API key
	bearerTokenParam = "bearertoken" // optional parameter for providing an access token
	sslModeParam     = "sslmode"     // optional parameter for providing a Dgraph SSL mode
	namespaceParam   = "namespace"   // optional parameter for providing a Dgraph namespace ID
	sslModeDisable   = "disable"
	sslModeRequire   = "require"
	sslModeVerifyCA  = "verify-ca"
)

type bearerCreds struct {
	token string
}

func (a *bearerCreds) GetRequestMetadata(ctx context.Context, uri ...string) (
	map[string]string, error) {

	return map[string]string{"Authorization": fmt.Sprintf("Bearer %s", a.token)}, nil
}

func (a *bearerCreds) RequireTransportSecurity() bool {
	return true
}

type clientOptions struct {
	namespace uint64
	gopts     []grpc.DialOption
	username  string
	password  string
}

// ClientOption is a function that modifies the client options.
type ClientOption func(*clientOptions) error

// WithDgraphAPIKey will use the provided API key for authentication for Dgraph Cloud.
func WithDgraphAPIKey(apiKey string) ClientOption {
	return func(o *clientOptions) error {
		o.gopts = append(o.gopts, grpc.WithPerRPCCredentials(&authCreds{token: apiKey}))
		return nil
	}
}

// WithBearerToken uses the provided token and presents it as a Bearer Token
// in the HTTP Authorization header for authentication against a Dgraph Cluster.
// This can be used to connect to Hypermode Cloud.
func WithBearerToken(token string) ClientOption {
	return func(o *clientOptions) error {
		o.gopts = append(o.gopts, grpc.WithPerRPCCredentials(&bearerCreds{token: token}))
		return nil
	}
}

func WithSkipTLSVerify() ClientOption {
	return func(o *clientOptions) error {
		o.gopts = append(o.gopts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
		return nil
	}
}

// WithSystemCertPool will use the system cert pool and setup a TLS connection with Dgraph cluster.
func WithSystemCertPool() ClientOption {
	return func(o *clientOptions) error {
		pool, err := x509.SystemCertPool()
		if err != nil {
			return fmt.Errorf("failed to create system cert pool: %w", err)
		}

		creds := credentials.NewClientTLSFromCert(pool, "")
		o.gopts = append(o.gopts, grpc.WithTransportCredentials(creds))
		return nil
	}
}

// WithNamespace logs into the given namespace.
func WithNamespace(nsID uint64) ClientOption {
	return func(o *clientOptions) error {
		o.namespace = nsID
		return nil
	}
}

// WithACLCreds will use the provided username and password for ACL authentication.
// If namespace is not provided, it logs into the galaxy namespace.
func WithACLCreds(username, password string) ClientOption {
	return func(o *clientOptions) error {
		o.username = username
		o.password = password
		return nil
	}
}

// WithResponseFormat sets the response format for queries. By default, the
// response format is JSON. We can also specify RDF format.
func WithResponseFormat(respFormat api.Request_RespFormat) TxnOption {
	return func(o *txnOptions) error {
		o.respFormat = respFormat
		return nil
	}
}

// WithGrpcOption will add a grpc.DialOption to the client.
// This is useful for setting custom  grpc options.
func WithGrpcOption(opt grpc.DialOption) ClientOption {
	return func(o *clientOptions) error {
		o.gopts = append(o.gopts, opt)
		return nil
	}
}

// Open creates a new Dgraph client by parsing a connection string of the form:
// dgraph://<optional-login>:<optional-password>@<host>:<port>?<optional-params>
// For example `dgraph://localhost:9080?sslmode=require`
//
// Parameters:
// - apikey: a Dgraph Cloud API key for authentication
// - bearertoken: a token for bearer authentication
// - sslmode: SSL connection mode (options: disable, require, verify-ca)
//   - disable: No TLS (default)
//   - require: Use TLS but skip certificate verification
//   - verify-ca: Use TLS and verify the certificate against system CA
//
// If credentials are provided, Open connects to the gRPC endpoint and authenticates the user.
// An error can be returned if the Dgraph cluster is not yet ready to accept requests--the text
// of the error in this case will contain the string "Please retry".
func Open(connStr string) (*Dgraph, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	params, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("malformed connection string: %w", err)
	}

	apiKey := params.Get(cloudAPIKeyParam)
	bearerToken := params.Get(bearerTokenParam)
	sslMode := params.Get(sslModeParam)
	nsID := params.Get(namespaceParam)

	if u.Scheme != dgraphScheme {
		return nil, fmt.Errorf("invalid scheme: must start with %s://", dgraphScheme)
	}
	if apiKey != "" && bearerToken != "" {
		return nil, errors.New("invalid connection string: both apikey and bearertoken cannot be provided")
	}
	if len(strings.Split(u.Host, ":")) != 2 {
		return nil, errors.New("invalid connection string: host url must have both host and port")
	}
	if strings.Split(u.Host, ":")[1] == "" {
		return nil, errors.New("invalid connection string: missing port after port-separator colon")
	}

	opts := []ClientOption{}
	if apiKey != "" {
		opts = append(opts, WithDgraphAPIKey(apiKey))
	}
	if bearerToken != "" {
		opts = append(opts, WithBearerToken(bearerToken))
	}

	if sslMode == "" {
		sslMode = sslModeDisable
	}
	switch sslMode {
	case sslModeDisable:
		opts = append(opts, WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	case sslModeRequire:
		opts = append(opts, WithSkipTLSVerify())
	case sslModeVerifyCA:
		opts = append(opts, WithSystemCertPool())
	default:
		return nil, fmt.Errorf("invalid SSL mode: %s (must be one of %s, %s, %s)",
			sslMode, sslModeDisable, sslModeRequire, sslModeVerifyCA)
	}

	if nsID != "" {
		nsID, err := strconv.ParseUint(nsID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid namespace ID: %w", err)
		}
		opts = append(opts, WithNamespace(nsID))
	}

	if u.User != nil {
		username := u.User.Username()
		password, _ := u.User.Password()
		if username == "" || password == "" {
			return nil, errors.New("invalid connection string: both username and password must be provided")
		}
		opts = append(opts, WithACLCreds(username, password))
	}

	return NewClient(u.Host, opts...)
}

// NewClient creates a new Dgraph client for a single endpoint.
// If ACL connection options are present, an login attempt is made
// using the supplied credentials.
func NewClient(endpoint string, opts ...ClientOption) (*Dgraph, error) {
	return NewRoundRobinClient([]string{endpoint}, opts...)
}

// NewRoundRobinClient creates a new Dgraph client for a list
// of endpoints. It will round robin among the provided endpoints.
// If ACL connection options are present, an login attempt is made
// using the supplied credentials.
func NewRoundRobinClient(endpoints []string, opts ...ClientOption) (*Dgraph, error) {
	co := &clientOptions{}
	for _, opt := range opts {
		if err := opt(co); err != nil {
			return nil, err
		}
	}

	conns := make([]*grpc.ClientConn, len(endpoints))
	dc := make([]api.DgraphClient, len(endpoints))
	for i, endpoint := range endpoints {
		conn, err := grpc.NewClient(endpoint, co.gopts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to endpoint [%s]: %w", endpoint, err)
		}
		conns[i] = conn
		dc[i] = api.NewDgraphClient(conn)
	}

	d := &Dgraph{dc: dc}
	if co.username != "" && co.password != "" {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		if err := d.login(ctx, co.username, co.password, co.namespace); err != nil {
			d.Close()
			return nil, fmt.Errorf("failed to sign in user: %w", err)
		}
	}

	if _, err := dc[0].CheckVersion(context.Background(), &api.Check{}); err != nil {
		d.Close()
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	return d, nil
}

// GetAPIClients returns the api.DgraphClient that is useful for advanced
// cases when grpc API that are not exposed in dgo needs to be used.
func (d *Dgraph) GetAPIClients() []api.DgraphClient {
	return d.dc
}

// Close shutdown down all the connections to the Dgraph Cluster.
func (d *Dgraph) Close() {
	for _, conn := range d.conns {
		_ = conn.Close()
	}
}

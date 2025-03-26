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
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/dgraph-io/dgo/v240/protos/api"
)

const (
	cloudPort = "443"

	dgraphScheme     = "dgraph"
	cloudAPIKeyParam = "apikey"      // optional parameter for providing a Dgraph Cloud API key
	bearerTokenParam = "bearertoken" // optional parameter for providing an access token
	sslModeParam     = "sslmode"     // optional parameter for providing a Dgraph SSL mode
	sslModeDisable   = "disable"
	sslModeRequire   = "require"
	sslModeVerifyCA  = "verify-ca"
)

// Dgraph is a transaction-aware client to a Dgraph cluster.
type Dgraph struct {
	jwtMutex sync.RWMutex
	jwt      api.Jwt

	conns []*grpc.ClientConn
	dc    []api.DgraphClient
}

type authCreds struct {
	token string
}

func (a *authCreds) GetRequestMetadata(ctx context.Context, uri ...string) (
	map[string]string, error) {

	return map[string]string{"Authorization": a.token}, nil
}

func (a *authCreds) RequireTransportSecurity() bool {
	return true
}

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
	gopts    []grpc.DialOption
	username string
	password string
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

// WithACLCreds will use the provided username and password for ACL authentication.
func WithACLCreds(username, password string) ClientOption {
	return func(o *clientOptions) error {
		o.username = username
		o.password = password
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

	if u.Scheme != dgraphScheme {
		return nil, fmt.Errorf("invalid scheme: must start with %s://", dgraphScheme)
	}
	if apiKey != "" && bearerToken != "" {
		return nil, errors.New("invalid connection string: both apikey and bearertoken cannot be provided")
	}
	if !strings.Contains(u.Host, ":") {
		return nil, errors.New("invalid connection string: host url must have both host and port")
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := d.Login(ctx, co.username, co.password); err != nil {
			d.Close()
			return nil, fmt.Errorf("failed to sign in user: %w", err)
		}
	}
	return d, nil
}

// Close shutdown down all the connections to the Dgraph Cluster.
func (d *Dgraph) Close() {
	for _, conn := range d.conns {
		_ = conn.Close()
	}
}

// NewDgraphClient creates a new Dgraph (client) for interacting with Alphas.
// The client is backed by multiple connections to the same or different
// servers in a cluster.
//
// A single Dgraph (client) is thread safe for sharing with multiple goroutines.
//
// Deprecated: Use dgo.NewClient or dgo.Open instead.
func NewDgraphClient(clients ...api.DgraphClient) *Dgraph {
	return &Dgraph{dc: clients}
}

// DialCloud creates a new TLS connection to a Dgraph Cloud backend
//
//	It requires the backend endpoint as well as the api token
//	Usage:
//		conn, err := dgo.DialCloud("CLOUD_ENDPOINT","API_TOKEN")
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer conn.Close()
//		dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
//
// Deprecated: Use dgo.NewClient or dgo.Open instead.
func DialCloud(endpoint, key string) (*grpc.ClientConn, error) {
	var grpcHost string
	switch {
	case strings.Contains(endpoint, ".grpc.") && strings.Contains(endpoint, ":"+cloudPort):
		// if we already have the grpc URL with the port, we don't need to do anything
		grpcHost = endpoint
	case strings.Contains(endpoint, ".grpc.") && !strings.Contains(endpoint, ":"+cloudPort):
		// if we have the grpc URL without the port, just add the port
		grpcHost = endpoint + ":" + cloudPort
	default:
		// otherwise, parse the non-grpc URL and add ".grpc." along with port to it.
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = "https://" + endpoint
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		urlParts := strings.SplitN(u.Host, ".", 2)
		if len(urlParts) < 2 {
			return nil, errors.New("invalid URL to Dgraph Cloud")
		}
		grpcHost = urlParts[0] + ".grpc." + urlParts[1] + ":" + cloudPort
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	creds := credentials.NewClientTLSFromCert(pool, "")
	return grpc.NewClient(
		grpcHost,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(&authCreds{key}),
	)
}

func (d *Dgraph) login(ctx context.Context, userid string, password string,
	namespace uint64) error {

	d.jwtMutex.Lock()
	defer d.jwtMutex.Unlock()

	dc := d.anyClient()
	loginRequest := &api.LoginRequest{
		Userid:    userid,
		Password:  password,
		Namespace: namespace,
	}
	resp, err := dc.Login(ctx, loginRequest)
	if err != nil {
		return err
	}

	return proto.Unmarshal(resp.Json, &d.jwt)
}

// GetJwt returns back the JWT for the dgraph client.
//
// Deprecated
func (d *Dgraph) GetJwt() api.Jwt {
	d.jwtMutex.RLock()
	defer d.jwtMutex.RUnlock()
	return d.jwt
}

// Login logs in the current client using the provided credentials into
// default namespace (0). Valid for the duration the client is alive.
func (d *Dgraph) Login(ctx context.Context, userid string, password string) error {
	return d.login(ctx, userid, password, 0)
}

// LoginIntoNamespace logs in the current client using the provided credentials.
// Valid for the duration the client is alive.
func (d *Dgraph) LoginIntoNamespace(ctx context.Context,
	userid string, password string, namespace uint64) error {

	return d.login(ctx, userid, password, namespace)
}

// Alter can be used to do the following by setting various fields of api.Operation:
//  1. Modify the schema.
//  2. Drop a predicate.
//  3. Drop the database.
func (d *Dgraph) Alter(ctx context.Context, op *api.Operation) error {
	dc := d.anyClient()
	_, err := dc.Alter(d.getContext(ctx), op)
	if isJwtExpired(err) {
		if err := d.retryLogin(ctx); err != nil {
			return err
		}
		_, err = dc.Alter(d.getContext(ctx), op)
	}
	return err
}

// Relogin relogin the current client using the refresh token. This can be used when the
// access-token gets expired.
func (d *Dgraph) Relogin(ctx context.Context) error {
	return d.retryLogin(ctx)
}

func (d *Dgraph) retryLogin(ctx context.Context) error {
	d.jwtMutex.Lock()
	defer d.jwtMutex.Unlock()

	if len(d.jwt.RefreshJwt) == 0 {
		return fmt.Errorf("refresh jwt should not be empty")
	}

	dc := d.anyClient()
	loginRequest := &api.LoginRequest{
		RefreshToken: d.jwt.RefreshJwt,
	}
	resp, err := dc.Login(ctx, loginRequest)
	if err != nil {
		return err
	}

	return proto.Unmarshal(resp.Json, &d.jwt)
}

func (d *Dgraph) getContext(ctx context.Context) context.Context {
	d.jwtMutex.RLock()
	defer d.jwtMutex.RUnlock()

	if len(d.jwt.AccessJwt) > 0 {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			// no metadata key is in the context, add one
			md = metadata.New(nil)
		}
		md.Set("accessJwt", d.jwt.AccessJwt)
		return metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

// isJwtExpired returns true if the error indicates that the jwt has expired.
func isJwtExpired(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	return ok && st.Code() == codes.Unauthenticated &&
		strings.Contains(err.Error(), "Token is expired")
}

func (d *Dgraph) anyClient() api.DgraphClient {
	//nolint:gosec
	return d.dc[rand.Intn(len(d.dc))]
}

// DeleteEdges sets the edges corresponding to predicates
// on the node with the given uid for deletion.
// This helper function doesn't run the mutation on the server.
// Txn needs to be committed in order to execute the mutation.
func DeleteEdges(mu *api.Mutation, uid string, predicates ...string) {
	for _, predicate := range predicates {
		mu.Del = append(mu.Del, &api.NQuad{
			Subject:   uid,
			Predicate: predicate,
			// _STAR_ALL is defined as x.Star in x package.
			ObjectValue: &api.Value{Val: &api.Value_DefaultVal{DefaultVal: "_STAR_ALL"}},
		})
	}
}

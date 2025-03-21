/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/dgraph-io/dgo/v240/protos/api"
	apiv25 "github.com/dgraph-io/dgo/v240/protos/api.v25"
)

const (
	cloudPort    = "443"
	dgraphScheme = "dgraph"
	// optional parameter for providing a Dgraph Cloud API key
	cloudAPIKeyParam = "apikey"
	// optional parameter for providing a Dgraph SSL mode
	sslModeParam    = "sslmode"
	sslModeDisable  = "disable"
	sslModeRequire  = "require"
	sslModeVerifyCA = "verify-ca"
)

// Dgraph is a transaction-aware client to a Dgraph cluster.
type Dgraph struct {
	jwtMutex sync.RWMutex
	jwt      api.Jwt

	conns []*grpc.ClientConn
	dc    []api.DgraphClient
	dcv25 []apiv25.DgraphClient
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

// Open creates a new Dgraph client by parsing a connection string of the form:
// dgraph://<optional-login>:<optional-password>@<host>:<port>?<optional-params>
// For example `dgraph://localhost:9080?sslmode=require`
// It connects to the gRPC endpoint and, if credentials are provided, signs in the user.
// An error can be returned if the Dgraph cluster is not yet ready to accept requests--the text
// of the error in this case will contain the string "Please retry".
func Open(connStr string) (*Dgraph, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	if u.Scheme != dgraphScheme {
		return nil, fmt.Errorf("invalid scheme: must start with %s://", dgraphScheme)
	}

	opts := []ClientOption{}

	apiKey := u.Query().Get(cloudAPIKeyParam)
	sslMode := u.Query().Get(sslModeParam)

	if apiKey != "" {
		opts = append(opts, WithDgraphAPIKey(apiKey))
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
		return nil, fmt.Errorf("invalid SSL mode: %s (must be one of %s, %s, %s)", sslMode, sslModeDisable, sslModeRequire, sslModeVerifyCA)
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
	dcv25 := make([]apiv25.DgraphClient, len(clients))
	for i, client := range clients {
		dcv25[i] = apiv25.NewDgraphClient(api.GetConn(client))
	}
	return &Dgraph{dc: clients, dcv25: dcv25}
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
//
// Deprecated: user SignInUser instead.
func (d *Dgraph) Login(ctx context.Context, userid string, password string) error {
	return d.login(ctx, userid, password, 0)
}

// LoginIntoNamespace logs in the current client using the provided credentials.
// Valid for the duration the client is alive.
//
// Deprecated: use SignInUser instead.
func (d *Dgraph) LoginIntoNamespace(ctx context.Context,
	userid string, password string, namespace uint64) error {

	return d.login(ctx, userid, password, namespace)
}

// Alter can be used to do the following by setting various fields of api.Operation:
//  1. Modify the schema.
//  2. Drop a predicate.
//  3. Drop the database.
//
// Deprecated: use DropAllNamespaces, DropAll, DropData, DropPredicate, DropType, SetSchema instead.
func (d *Dgraph) Alter(ctx context.Context, op *api.Operation) error {
	dc := d.anyClient()
	_, err := doWithRetryLogin(ctx, d, func() (*api.Payload, error) {
		return dc.Alter(d.getContext(ctx), op)
	})
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

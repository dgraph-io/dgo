/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/dgraph-io/dgo/v240/protos/api"
	apiv25 "github.com/dgraph-io/dgo/v240/protos/api.v25"
)

const (
	RootNamespace = "root"
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
	gopts              []grpc.DialOption
	username, password string
}

// ClientOption is a function that modifies the client options.
type ClientOption func(*clientOptions) error

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

func WithSkipTLSVerify() ClientOption {
	return func(o *clientOptions) error {
		o.gopts = append(o.gopts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
		return nil
	}
}

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

// WithGrpcOption will add a grpc.DialOption to the client.
// This is useful for setting custom  grpc options.
func WithGrpcOption(opt grpc.DialOption) ClientOption {
	return func(o *clientOptions) error {
		o.gopts = append(o.gopts, opt)
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
	dcv25 := make([]apiv25.DgraphClient, len(endpoints))
	for i, endpoint := range endpoints {
		conn, err := grpc.NewClient(endpoint, co.gopts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to endpoint [%s]: %w", endpoint, err)
		}
		conns[i] = conn
		dc[i] = api.NewDgraphClient(conn)
		dcv25[i] = apiv25.NewDgraphClient(conn)
	}

	d := &Dgraph{dc: dc, dcv25: dcv25}
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

func (d *Dgraph) anyClientv25() apiv25.DgraphClient {
	//nolint:gosec
	return d.dcv25[rand.Intn(len(d.dcv25))]
}

// CreateNamespace creates a new namespace with the given name and password for groot user.
func (d *Dgraph) CreateNamespace(ctx context.Context, name string) error {
	dc := d.anyClientv25()
	req := &apiv25.CreateNamespaceRequest{NsName: name}
	_, err := doWithRetryLogin(ctx, d, func() (*apiv25.CreateNamespaceResponse, error) {
		return dc.CreateNamespace(d.getContext(ctx), req)
	})
	return err
}

// DropNamespace deletes the namespace with the given name.
func (d *Dgraph) DropNamespace(ctx context.Context, name string) error {
	dc := d.anyClientv25()
	req := &apiv25.DropNamespaceRequest{NsName: name}
	_, err := doWithRetryLogin(ctx, d, func() (*apiv25.DropNamespaceResponse, error) {
		return dc.DropNamespace(d.getContext(ctx), req)
	})
	return err
}

// RenameNamespace renames the namespace from the given name to the new name.
func (d *Dgraph) RenameNamespace(ctx context.Context, from string, to string) error {
	dc := d.anyClientv25()
	req := &apiv25.UpdateNamespaceRequest{NsName: from, RenameToNs: to}
	_, err := doWithRetryLogin(ctx, d, func() (*apiv25.UpdateNamespaceResponse, error) {
		return dc.UpdateNamespace(d.getContext(ctx), req)
	})
	return err
}

// ListNamespaces returns a map of namespace names to their details.
func (d *Dgraph) ListNamespaces(ctx context.Context) (map[string]*apiv25.Namespace, error) {
	dc := d.anyClientv25()
	resp, err := doWithRetryLogin(ctx, d, func() (*apiv25.ListNamespacesResponse, error) {
		return dc.ListNamespaces(d.getContext(ctx), &apiv25.ListNamespacesRequest{})
	})
	if err != nil {
		return nil, err
	}

	return resp.NsList, nil
}

func doWithRetryLogin[T any](ctx context.Context, d *Dgraph, f func() (*T, error)) (*T, error) {
	resp, err := f()
	if isJwtExpired(err) {
		if err := d.retryLogin(ctx); err != nil {
			return nil, err
		}
		return f()
	}
	return resp, err
}

// SignInUser logs the user in using the provided username and password.
func (d *Dgraph) SignInUser(ctx context.Context, username, password string) error {
	d.jwtMutex.Lock()
	defer d.jwtMutex.Unlock()

	dc := d.anyClientv25()
	req := &apiv25.SignInUserRequest{UserId: username, Password: password}
	resp, err := dc.SignInUser(ctx, req)
	if err != nil {
		return err
	}

	d.jwt.AccessJwt = resp.AccessJwt
	d.jwt.RefreshJwt = resp.RefreshJwt
	return nil
}

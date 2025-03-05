//
// SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
// SPDX-License-Identifier: Apache-2.0

// Style guide for Protocol Buffer 3.
// Use CamelCase (with an initial capital) for message names – for example,
// SongServerRequest. Use underscore_separated_names for field names – for
// example, song_name.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.12
// source: api.v25.proto

package api_v25

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Dgraph_Ping_FullMethodName            = "/api.v25.Dgraph/Ping"
	Dgraph_SignInUser_FullMethodName      = "/api.v25.Dgraph/SignInUser"
	Dgraph_Alter_FullMethodName           = "/api.v25.Dgraph/Alter"
	Dgraph_RunDQL_FullMethodName          = "/api.v25.Dgraph/RunDQL"
	Dgraph_CreateNamespace_FullMethodName = "/api.v25.Dgraph/CreateNamespace"
	Dgraph_DropNamespace_FullMethodName   = "/api.v25.Dgraph/DropNamespace"
	Dgraph_UpdateNamespace_FullMethodName = "/api.v25.Dgraph/UpdateNamespace"
	Dgraph_ListNamespaces_FullMethodName  = "/api.v25.Dgraph/ListNamespaces"
)

// DgraphClient is the client API for Dgraph service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DgraphClient interface {
	Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error)
	SignInUser(ctx context.Context, in *SignInUserRequest, opts ...grpc.CallOption) (*SignInUserResponse, error)
	Alter(ctx context.Context, in *AlterRequest, opts ...grpc.CallOption) (*AlterResponse, error)
	RunDQL(ctx context.Context, in *RunDQLRequest, opts ...grpc.CallOption) (*RunDQLResponse, error)
	CreateNamespace(ctx context.Context, in *CreateNamespaceRequest, opts ...grpc.CallOption) (*CreateNamespaceResponse, error)
	DropNamespace(ctx context.Context, in *DropNamespaceRequest, opts ...grpc.CallOption) (*DropNamespaceResponse, error)
	UpdateNamespace(ctx context.Context, in *UpdateNamespaceRequest, opts ...grpc.CallOption) (*UpdateNamespaceResponse, error)
	ListNamespaces(ctx context.Context, in *ListNamespacesRequest, opts ...grpc.CallOption) (*ListNamespacesResponse, error)
}

type dgraphClient struct {
	cc grpc.ClientConnInterface
}

func NewDgraphClient(cc grpc.ClientConnInterface) DgraphClient {
	return &dgraphClient{cc}
}

func (c *dgraphClient) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error) {
	out := new(PingResponse)
	err := c.cc.Invoke(ctx, Dgraph_Ping_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dgraphClient) SignInUser(ctx context.Context, in *SignInUserRequest, opts ...grpc.CallOption) (*SignInUserResponse, error) {
	out := new(SignInUserResponse)
	err := c.cc.Invoke(ctx, Dgraph_SignInUser_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dgraphClient) Alter(ctx context.Context, in *AlterRequest, opts ...grpc.CallOption) (*AlterResponse, error) {
	out := new(AlterResponse)
	err := c.cc.Invoke(ctx, Dgraph_Alter_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dgraphClient) RunDQL(ctx context.Context, in *RunDQLRequest, opts ...grpc.CallOption) (*RunDQLResponse, error) {
	out := new(RunDQLResponse)
	err := c.cc.Invoke(ctx, Dgraph_RunDQL_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dgraphClient) CreateNamespace(ctx context.Context, in *CreateNamespaceRequest, opts ...grpc.CallOption) (*CreateNamespaceResponse, error) {
	out := new(CreateNamespaceResponse)
	err := c.cc.Invoke(ctx, Dgraph_CreateNamespace_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dgraphClient) DropNamespace(ctx context.Context, in *DropNamespaceRequest, opts ...grpc.CallOption) (*DropNamespaceResponse, error) {
	out := new(DropNamespaceResponse)
	err := c.cc.Invoke(ctx, Dgraph_DropNamespace_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dgraphClient) UpdateNamespace(ctx context.Context, in *UpdateNamespaceRequest, opts ...grpc.CallOption) (*UpdateNamespaceResponse, error) {
	out := new(UpdateNamespaceResponse)
	err := c.cc.Invoke(ctx, Dgraph_UpdateNamespace_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dgraphClient) ListNamespaces(ctx context.Context, in *ListNamespacesRequest, opts ...grpc.CallOption) (*ListNamespacesResponse, error) {
	out := new(ListNamespacesResponse)
	err := c.cc.Invoke(ctx, Dgraph_ListNamespaces_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DgraphServer is the server API for Dgraph service.
// All implementations must embed UnimplementedDgraphServer
// for forward compatibility
type DgraphServer interface {
	Ping(context.Context, *PingRequest) (*PingResponse, error)
	SignInUser(context.Context, *SignInUserRequest) (*SignInUserResponse, error)
	Alter(context.Context, *AlterRequest) (*AlterResponse, error)
	RunDQL(context.Context, *RunDQLRequest) (*RunDQLResponse, error)
	CreateNamespace(context.Context, *CreateNamespaceRequest) (*CreateNamespaceResponse, error)
	DropNamespace(context.Context, *DropNamespaceRequest) (*DropNamespaceResponse, error)
	UpdateNamespace(context.Context, *UpdateNamespaceRequest) (*UpdateNamespaceResponse, error)
	ListNamespaces(context.Context, *ListNamespacesRequest) (*ListNamespacesResponse, error)
	mustEmbedUnimplementedDgraphServer()
}

// UnimplementedDgraphServer must be embedded to have forward compatible implementations.
type UnimplementedDgraphServer struct {
}

func (UnimplementedDgraphServer) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedDgraphServer) SignInUser(context.Context, *SignInUserRequest) (*SignInUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignInUser not implemented")
}
func (UnimplementedDgraphServer) Alter(context.Context, *AlterRequest) (*AlterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Alter not implemented")
}
func (UnimplementedDgraphServer) RunDQL(context.Context, *RunDQLRequest) (*RunDQLResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RunDQL not implemented")
}
func (UnimplementedDgraphServer) CreateNamespace(context.Context, *CreateNamespaceRequest) (*CreateNamespaceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateNamespace not implemented")
}
func (UnimplementedDgraphServer) DropNamespace(context.Context, *DropNamespaceRequest) (*DropNamespaceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DropNamespace not implemented")
}
func (UnimplementedDgraphServer) UpdateNamespace(context.Context, *UpdateNamespaceRequest) (*UpdateNamespaceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateNamespace not implemented")
}
func (UnimplementedDgraphServer) ListNamespaces(context.Context, *ListNamespacesRequest) (*ListNamespacesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListNamespaces not implemented")
}
func (UnimplementedDgraphServer) mustEmbedUnimplementedDgraphServer() {}

// UnsafeDgraphServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DgraphServer will
// result in compilation errors.
type UnsafeDgraphServer interface {
	mustEmbedUnimplementedDgraphServer()
}

func RegisterDgraphServer(s grpc.ServiceRegistrar, srv DgraphServer) {
	s.RegisterService(&Dgraph_ServiceDesc, srv)
}

func _Dgraph_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DgraphServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Dgraph_Ping_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DgraphServer).Ping(ctx, req.(*PingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Dgraph_SignInUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignInUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DgraphServer).SignInUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Dgraph_SignInUser_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DgraphServer).SignInUser(ctx, req.(*SignInUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Dgraph_Alter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AlterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DgraphServer).Alter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Dgraph_Alter_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DgraphServer).Alter(ctx, req.(*AlterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Dgraph_RunDQL_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunDQLRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DgraphServer).RunDQL(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Dgraph_RunDQL_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DgraphServer).RunDQL(ctx, req.(*RunDQLRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Dgraph_CreateNamespace_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateNamespaceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DgraphServer).CreateNamespace(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Dgraph_CreateNamespace_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DgraphServer).CreateNamespace(ctx, req.(*CreateNamespaceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Dgraph_DropNamespace_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DropNamespaceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DgraphServer).DropNamespace(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Dgraph_DropNamespace_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DgraphServer).DropNamespace(ctx, req.(*DropNamespaceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Dgraph_UpdateNamespace_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateNamespaceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DgraphServer).UpdateNamespace(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Dgraph_UpdateNamespace_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DgraphServer).UpdateNamespace(ctx, req.(*UpdateNamespaceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Dgraph_ListNamespaces_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListNamespacesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DgraphServer).ListNamespaces(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Dgraph_ListNamespaces_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DgraphServer).ListNamespaces(ctx, req.(*ListNamespacesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Dgraph_ServiceDesc is the grpc.ServiceDesc for Dgraph service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Dgraph_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.v25.Dgraph",
	HandlerType: (*DgraphServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _Dgraph_Ping_Handler,
		},
		{
			MethodName: "SignInUser",
			Handler:    _Dgraph_SignInUser_Handler,
		},
		{
			MethodName: "Alter",
			Handler:    _Dgraph_Alter_Handler,
		},
		{
			MethodName: "RunDQL",
			Handler:    _Dgraph_RunDQL_Handler,
		},
		{
			MethodName: "CreateNamespace",
			Handler:    _Dgraph_CreateNamespace_Handler,
		},
		{
			MethodName: "DropNamespace",
			Handler:    _Dgraph_DropNamespace_Handler,
		},
		{
			MethodName: "UpdateNamespace",
			Handler:    _Dgraph_UpdateNamespace_Handler,
		},
		{
			MethodName: "ListNamespaces",
			Handler:    _Dgraph_ListNamespaces_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.v25.proto",
}

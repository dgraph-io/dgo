/*
 * Copyright (C) 2017 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package test

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/stretchr/testify/require"
)

func TestAcl(t *testing.T) {
	// setup a cluster
	//cluster := setupCluster()
	//defer cluster.Close()
	dg, close := GetDgraphClient()
	defer close()

	createAccountAndData(t, dg)

	// try to query the user whose name is alice
	ctx := context.Background()
	if err := dg.Login(ctx, user, password); err != nil {
		t.Fatalf("unable to login using the account %v", user)
	}

	ctxWithUserJwt := dg.GetContext(ctx)
	require.True(t, ctxWithUserJwt.Value("accessJwt") != nil, "the accessJwt "+
		"should not be empty")

	txn := dg.NewTxn()
	const cityQuery = `
	{
		q(func: eq(city_name, "SF")) {
			name
		}
	}`

	txn = dg.NewTxn()
	_, err := txn.Query(ctxWithUserJwt, cityQuery)

	// verify that the access is not authorized
	require.Error(t, err)
}

var user = "alice"
var password = "password123"
var rootDir = filepath.Join(os.TempDir(), "acl_test")

func createAccountAndData(t *testing.T, dg *dgo.Dgraph) {
	ctx := context.Background()
	if err := dg.Login(ctx, "admin", "password"); err != nil {
		t.Fatalf("unable to login using the admin account")
	}

	ctxWithAdminJwt := dg.GetContext(ctx)
	op := api.Operation{
		DropAll: true,
	}
	if err := dg.Alter(ctxWithAdminJwt, &op); err != nil {
		t.Fatalf("Unable to cleanup db:%v", err)
	}

	// use commands to create users and groups
	createUserCmd := exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"acl", "useradd",
		"-d", "localhost:9180",
		"-u", user, "-p", password, "--adminPassword", "password")
	if err := createUserCmd.Run(); err != nil {
		t.Fatalf("Unable to create user:%v", err)
	}

	// create some data, e.g. user with name alice
	require.NoError(t, dg.Alter(ctxWithAdminJwt, &api.Operation{
		Schema: `city_name: string @index(exact) .`,
	}))

	txn := dg.NewTxn()
	_, err := txn.Mutate(ctxWithAdminJwt, &api.Mutation{
		SetNquads: []byte(`
			_:a <city_name> "SF" .
		`),
	})
	require.NoError(t, err)
	require.NoError(t, txn.Commit(ctx))
}

func setupCluster() *DgraphCluster {
	if err := MakeDirEmpty(rootDir); err != nil {
		log.Fatalf("Unable to create dir %v", rootDir)
	}

	cluster := NewDgraphCluster(rootDir)
	if err := cluster.Start(); err != nil {
		log.Fatalf("Unable to start cluster: %v", err)
	}
	return cluster
}

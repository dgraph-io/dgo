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
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAcl(t *testing.T) {
	// setup a cluster and clean the database
	cluster := setupCluster()
	defer cluster.Close()
	op := api.Operation{
		DropAll: true,
	}
	ctx := context.Background()
	if err := cluster.Client.Alter(ctx, &op); err != nil {
		t.Fatalf("Unable to cleanup db:%v", err)
	}

	// use commands to create users and groups
	createUserCmd := exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"acl", "useradd",
		"-d", "localhost:" + cluster.DgraphPort,
		"-u", user, "-p", password)
	if err := createUserCmd.Run(); err != nil {
		t.Fatalf("Unable to create user:%v", err)
	}

	// create some data, e.g. user with name alice
	require.NoError(t, cluster.Client.Alter(ctx, &api.Operation{
		Schema: `name: string @index(exact) .`,
	}))

	txn := cluster.Client.NewTxn()
	_, err := txn.Mutate(ctx, &api.Mutation{
		SetNquads: []byte(`
			_:a <name> "Alice" .
		`),
	})
	require.NoError(t, err)
	require.NoError(t, txn.Commit(ctx))

	// try to query the user whose name is alice
	const aliceQuery = `
	{
		q(func: eq(name, "Alice")) {
			name
		}
	}`

	txn = cluster.Client.NewTxn()
	_, err = txn.Query(ctx, aliceQuery)

	// verify that the access is not authorized
	require.Error(t, err)
}

var user = "alice"
var password = "password123"
var rootDir = filepath.Join(os.TempDir(), "acl_test")

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


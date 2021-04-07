/*
 * Copyright (C) 2021 Dgraph Labs, Inc. and Contributors
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

package dgo_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v210/protos/api"
)

// LoginParams stores the information needed to perform a login request.
type LoginParams struct {
	Endpoint   string
	UserID     string
	Passwd     string
	Namespace  uint64
	RefreshJwt string
}

type GraphQLParams struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type HttpToken struct {
	UserId       string
	Password     string
	AccessJwt    string
	RefreshToken string
}

type GraphQLResponse struct {
	Data       json.RawMessage        `json:"data,omitempty"`
	Errors     GqlErrorList           `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type GqlErrorList []*GqlError

type GqlError struct {
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func MakeGQLRequestHelper(t *testing.T, endpoint string, params *GraphQLParams,
	token *HttpToken) *GraphQLResponse {

	b, err := json.Marshal(params)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token.AccessJwt != "" {
		req.Header.Set("X-Dgraph-AccessToken", token.AccessJwt)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var gqlResp GraphQLResponse
	err = json.Unmarshal(b, &gqlResp)
	require.NoError(t, err)

	return &gqlResp
}

func (errList GqlErrorList) Error() string {
	var buf bytes.Buffer
	for i, gqlErr := range errList {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(gqlErr.Error())
	}
	return buf.String()
}

func (gqlErr *GqlError) Error() string {
	var buf bytes.Buffer
	if gqlErr == nil {
		return ""
	}

	buf.WriteString(gqlErr.Message)
	return buf.String()
}
func MakeGQLRequest(t *testing.T, endpoint string, params *GraphQLParams,
	token *HttpToken) *GraphQLResponse {
	resp := MakeGQLRequestHelper(t, endpoint, params, token)
	if len(resp.Errors) == 0 || !strings.Contains(resp.Errors.Error(), "Token is expired") {
		return resp
	}
	var err error
	token, err = HttpLogin(&LoginParams{
		Endpoint:   endpoint,
		UserID:     token.UserId,
		Passwd:     token.Password,
		RefreshJwt: token.RefreshToken,
	})
	require.NoError(t, err)
	return MakeGQLRequestHelper(t, endpoint, params, token)
}

// HttpLogin sends a HTTP request to the server
// and returns the access JWT and refresh JWT extracted from
// the HTTP response
func HttpLogin(params *LoginParams) (*HttpToken, error) {
	loginPayload := api.LoginRequest{}
	if len(params.RefreshJwt) > 0 {
		loginPayload.RefreshToken = params.RefreshJwt
	} else {
		loginPayload.Userid = params.UserID
		loginPayload.Password = params.Passwd
	}

	login := `mutation login($userId: String, $password: String, $namespace: Int, $refreshToken: String) {
		login(userId: $userId, password: $password, namespace: $namespace, refreshToken: $refreshToken) {
			response {
				accessJWT
				refreshJWT
			}
		}
	}`

	gqlParams := GraphQLParams{
		Query: login,
		Variables: map[string]interface{}{
			"userId":       params.UserID,
			"password":     params.Passwd,
			"namespace":    params.Namespace,
			"refreshToken": params.RefreshJwt,
		},
	}
	body, err := json.Marshal(gqlParams)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to marshal body")
	}

	req, err := http.NewRequest("POST", params.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "login through curl failed")
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read from response")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("got non 200 response from the server with %s ",
			string(respBody)))
	}
	var outputJson map[string]interface{}
	if err := json.Unmarshal(respBody, &outputJson); err != nil {
		var errOutputJson map[string]interface{}
		if err := json.Unmarshal(respBody, &errOutputJson); err == nil {
			if _, ok := errOutputJson["errors"]; ok {
				return nil, errors.Errorf("response error: %v", string(respBody))
			}
		}
		return nil, errors.Wrapf(err, "unable to unmarshal the output to get JWTs")
	}

	data, found := outputJson["data"].(map[string]interface{})
	if !found {
		return nil, errors.Wrapf(err, "data entry found in the output")
	}

	l, found := data["login"].(map[string]interface{})
	if !found {
		return nil, errors.Wrapf(err, "data entry found in the output")
	}

	response, found := l["response"].(map[string]interface{})
	if !found {
		return nil, errors.Wrapf(err, "data entry found in the output")
	}

	newAccessJwt, found := response["accessJWT"].(string)
	if !found || newAccessJwt == "" {
		return nil, errors.Errorf("no access JWT found in the output")
	}
	newRefreshJwt, found := response["refreshJWT"].(string)
	if !found || newRefreshJwt == "" {
		return nil, errors.Errorf("no refresh JWT found in the output")
	}

	return &HttpToken{
		UserId:       params.UserID,
		Password:     params.Passwd,
		AccessJwt:    newAccessJwt,
		RefreshToken: newRefreshJwt,
	}, nil
}

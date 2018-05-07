// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build unit

package auth_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

type auth0RestTestConfig struct {
	hostID string
	poolID string
	admin  bool
	dfs    bool
	exp    time.Duration
	method string
	uri    string
}

var (
	originalAuth0TokenGetter         = auth.AuthTokenGetter
	originalAuth0RestTokenExpiration = auth.RestTokenExpiration
)

func BuildAuth0RestToken(r *http.Request) (string, error) {
	now := jwt.TimeFunc().UTC()
	iss := "https://my-issuer-domain.foo"
	aud := []string{"https://aud1.foo.bar", "https://audience2.other.thing/something"}
	iat := now.Unix()
	exp := now.Add(auth.RestTokenExpiration).Unix()
	claims := &auth.jwtAuth0Claims{iss, iat, exp, aud}
	restToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	delegatePrivKey, err := auth.getDelegatePrivateKey()
	if err != nil {
		return "", err
	}
	signedToken, err := restToken.SignedString(delegatePrivKey)
	return signedToken, err
}

func auth0RestTestCleanup() {
	auth.AuthTokenGetter = originalAuth0TokenGetter
	auth.RestTokenExpiration = originalAuth0RestTokenExpiration
}

func newAuth0TestConfig() auth0RestTestConfig {
	return auth0RestTestConfig{"mockHost", "mockPool", true, true, time.Minute, "GET", "/super/fake/request"}
}

func (s *TestAuthSuite) TestBuildAndExtractAuth0RestToken(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	// Create Rest Token
	restToken, err := BuildAuth0RestToken(req)
	c.Assert(err, IsNil)
	c.Assert(restToken, NotNil)

	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	h := req.Header.Get("Authorization")
	c.Assert(h, DeepEquals, fmt.Sprintf("Bearer %s", restToken))

	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	c.Assert(extractedToken, DeepEquals, restToken)

	// Parse token
	parsedToken, err := auth.ParseRestToken(extractedToken)
	c.Assert(err, IsNil)
	c.Assert(parsedToken, NotNil)
	c.Assert(parsedToken.RestToken(), DeepEquals, restToken)
	c.Assert(parsedToken.AuthToken(), DeepEquals, authToken)
	c.Assert(parsedToken.HasAdminAccess(), Equals, cfg.admin)
	c.Assert(parsedToken.Expired(), Equals, false)
	c.Assert(parsedToken.Valid(), IsNil)
	c.Assert(parsedToken.ValidateRequestHash(req), Equals, true)

	restTestCleanup()
}

func (s *TestAuthSuite) TestExpiredAuth0RestToken(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	auth.RestTokenExpiration = -1 * time.Hour
	// Create Rest Token
	restToken, err := BuildAuth0RestToken(req)
	c.Assert(err, IsNil)
	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	// Parse token
	_, err = auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrRestTokenExpired)

	restTestCleanup()
}

func (s *TestAuthSuite) TestTamperedAuth0RestToken(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	// Create Rest Token
	restToken, err := BuildAuth0RestToken(req)
	c.Assert(err, IsNil)
	// modify token
	l := len(restToken)
	restToken = restToken[:l-4] + "HOLA"
	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	c.Assert(extractedToken, DeepEquals, restToken)
	// Parse token
	_, err = auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrRestTokenBadSig)

	restTestCleanup()
}

func (s *TestAuthSuite) TestInvalidAuth0RestToken(c *C) {
	cfg := newTestConfig()
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	// Empty token
	auth.AddRestTokenToRequest(req, "")
	_, err := auth.ExtractRestToken(req)
	c.Assert(err, Equals, auth.ErrBadRestToken)

	// Invalid token
	req, _ = http.NewRequest(cfg.method, cfg.uri, nil)
	invalidToken := "THIS ISNT A REST TOKEN"
	auth.AddRestTokenToRequest(req, invalidToken)
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	token, err := auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrBadRestToken)
	c.Assert(token, IsNil)

	restTestCleanup()
}

func (s *TestAuthSuite) TestValidateAuth0RequestHash(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest("GET", cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	// Create Rest Token
	restToken, err := BuildAuth0RestToken(req)
	auth.AddRestTokenToRequest(req, restToken)
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	token, err := auth.ParseRestToken(extractedToken)
	c.Assert(err, IsNil)
	c.Assert(token.ValidateRequestHash(req), Equals, true)
	req.Method = "POST"
	c.Assert(token.ValidateRequestHash(req), Equals, false)

	restTestCleanup()
}

func (s *TestAuthSuite) TestExpiredAuth0AuthToken(c *C) {
	cfg := newTestConfig()
	cfg.exp = -1 * time.Hour
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	// Create Rest Token
	restToken, err := BuildAuth0RestToken(req)
	c.Assert(err, IsNil)
	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	// Parse token
	_, err = auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrIdentityTokenExpired)

	restTestCleanup()
}

func (s *TestAuthSuite) TestTamperedAuth0AuthToken(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	authToken = authToken[:40] + "HOLA" + authToken[44:]
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	// Create Rest Token
	restToken, err := BuildAuth0RestToken(req)
	c.Assert(err, IsNil)
	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	// Parse token
	_, err = auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrIdentityTokenBadSig)

	restTestCleanup()
}

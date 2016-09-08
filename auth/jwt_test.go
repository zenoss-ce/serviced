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
	"time"

	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

func (s *TestAuthSuite) TestIdentityHappyPath(c *C) {
	pubkey, _ := auth.RSAPublicKeyFromPEM(auth.DevPubKeyPEM)
	privkey, _ := auth.RSAPrivateKeyFromPEM(auth.DevPrivKeyPEM)
	token, err := auth.CreateJWTIdentity("host", "pool", true, false, pubkey, time.Minute, privkey)

	c.Assert(err, IsNil)

	identity, err := auth.ParseJWTIdentity(token, pubkey)
	c.Assert(err, IsNil)

	c.Assert(identity.HostID(), Equals, "host")
	c.Assert(identity.PoolID(), Equals, "pool")
	c.Assert(identity.Expired(), Equals, false)
	c.Assert(identity.HasAdminAccess(), Equals, true)
	c.Assert(identity.HasDFSAccess(), Equals, false)

	signer, _ := auth.RSASigner(privkey)
	message := []byte("this is a message")
	sig, _ := signer.Sign(message)

	verifier, err := identity.Verifier()
	c.Assert(err, IsNil)

	err = verifier.Verify(message, sig)
	c.Assert(err, IsNil)
}

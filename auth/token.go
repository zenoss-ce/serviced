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

package auth

import (
	"sync"
	"time"
)

var (
	currentToken string
	expiration   time.Time
	cond         = &sync.Cond{L: &sync.Mutex{}}
)

func now() time.Time {
	return time.Now().UTC()
}

func expired() bool {
	if currentToken == "" {
		return true
	}
	if expiration.IsZero() {
		return false
	}
	return expiration.Add(-expirationDelta).Before(now())
}

const (
	// expirationDelta is a margin of error during which a token should be
	// considered expired. This should help avoid expiration races when server
	// times don't match
	expirationDelta = 10 * time.Second
)

// TokenFunc is a function that can return an authentication token and its
// expiration time
type TokenFunc func() (string, int64, error)

// RefreshToken gets a new token, sets it as the current, and returns the expiration time
func RefreshToken(f TokenFunc) (int64, error) {
	log.Debug("Refreshing authentication token")
	token, expires, err := f()
	if err != nil {
		return 0, err
	}
	updateToken(token, expires)
	log.WithField("expiration", expires).Info("Received new authentication token")
	return expires, err
}

// AuthToken returns an unexpired auth token, blocking if necessary until
// authenticated
func AuthToken() string {
	cond.L.Lock()
	defer cond.L.Unlock()
	for expired() {
		cond.Wait()
	}
	return currentToken
}

func TokenLoop(f TokenFunc, done chan interface{}) {
	for {
		expires, err := RefreshToken(f)
		if err != nil {
			log.WithError(err).Warn("Unable to obtain authentication token. Retrying in 10s")
			select {
			case <-done:
				return
			case <-time.After(10 * time.Second):
			}
			continue
		}
		// Reauthenticate 1 minute before the token expires
		expiration := time.Unix(expires, 0).Sub(time.Now().UTC())
		refresh := expiration - time.Duration(1*time.Minute)
		select {
		case <-done:
			return
		case <-time.After(refresh):
		}
	}
}

func updateToken(token string, expires int64) {
	cond.L.Lock()
	currentToken = token
	expiration = time.Unix(expires, 0)
	cond.L.Unlock()
	cond.Broadcast()
}

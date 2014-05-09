// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// ConvertUp & NewUUID62 taken from https://raw.githubusercontent.com/xhroot/Koderank/48db8afb0759a354bcb16759e2769d3b7621769e/uuid/uuid.go
// under MIT License
/*
Copyright (c) 2012, Antonio Rodriguez <dev@xhroot.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package utils

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
)

const base62alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var randSource io.Reader = randReadT{}

// randReadT is a struct that implements the Reader interface and return random bytes
type randReadT struct{}

func (r randReadT) Read(p []byte) (n int, err error) {
	return rand.Read(p)
}

// NewUUID generate a new UUID
func NewUUID() (string, error) {
	b := make([]byte, 16)
	_, err := randSource.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// NewUUID62 createa a base-62 UUID.
func NewUUID62() (string, error) {
	b := make([]byte, 16)
	_, err := randSource.Read(b)
	if err != nil {
		return "", err
	}
	s := fmt.Sprintf("%x", b)
	return ConvertUp(s, base62alphabet), nil
}

// ConvertUp converts a hexadecimal UUID string to a base alphabet greater than
// 16. It is used here to compress a 32 character UUID down to 23 URL friendly
// characters.
func ConvertUp(oldNumber string, baseAlphabet string) string {
	n := big.NewInt(0)
	n.SetString(oldNumber, 16)

	base := big.NewInt(int64(len(baseAlphabet)))

	newNumber := make([]byte, 23) //converted size of max base-62 uuid
	i := len(newNumber)

	for n.Int64() != 0 {
		i--
		_, r := n.DivMod(n, base, big.NewInt(0))
		newNumber[i] = baseAlphabet[r.Int64()]
	}
	return string(newNumber[i:])
}

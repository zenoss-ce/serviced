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
	"bytes"
	"errors"
	"io"
<<<<<<< Updated upstream
	"net"
=======

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/proxy"
>>>>>>> Stashed changes
)

/*
   When establishing a connection to the mux, in addition to the address of the receiver,
   the sender sends an authentication token and signs the whole message. The token determines
   if the sender is authorized to send data to the receiver or not

   ---------------------------------------------------------------------------------------------------------
   | Auth Token length (4 bytes)  |     Auth Token (N bytes)  | Address (6 bytes) |  Signature (256 bytes) |
   ---------------------------------------------------------------------------------------------------------
*/

const (
	ADDRESS_BYTES = 6
)

var (
	ErrBadMuxAddress = errors.New("Bad mux address")
	ErrBadMuxHeader  = errors.New("Bad mux header")
<<<<<<< Updated upstream
)

func BuildAuthMuxHeader(address []byte, token string) ([]byte, error) {
	if len(address) != ADDRESS_BYTES {
		return nil, ErrBadMuxAddress
	}
	headerBuf := new(bytes.Buffer)

	// add token length
	var tokenLen uint32 = uint32(len(token))
	tokenLenBuf := make([]byte, 4)
	endian.PutUint32(tokenLenBuf, tokenLen)
	headerBuf.Write(tokenLenBuf)

	// add token
	headerBuf.Write([]byte(token))
=======
	ErrBadToken      = errors.New("Could not extract token")
)

const (
	SignatureLenBytes = 256
)

type Signature [SignatureLenBytes]byte
>>>>>>> Stashed changes

func (s Signature) ReadFrom(r io.Reader) (n int64, err error) {
	return io.ReadFull(r, s[:])
}

func (s Signature) WriteTo(w io.WriteTo) (n int64, err error) {
	return w.Write(s[:])
}

func (s Signature) Bytes() []byte {
	return s[:]
}

func init() {
	log.SetLevel(logrus.DebugLevel, true)
}

// Write writes the mux auth header to the stream
func WriteMuxHeader(w io.Writer, address proxy.MuxHeader, token Token) error {
	// set up a multi writer, for signing
	buf := &bytes.Buffer{}
	multi := io.MultiWriter(w, buf)

	// write the token
	if _, err := token.WriteTo(multi); err != nil {
		log.WithError(err).Debug("Could not write token")
		return err
	}
	log.Debug("Wrote the token")

	// write the mux header address
	if _, err := address.WriteTo(multi); err != nil {
		log.WithError(err).Debug("Could not write mux header address")
		return err
	}
	log.Debug("Wrote the mux header address")

<<<<<<< Updated upstream
	// Next tokeLen bytes contain the token
	token := string(rawHeader[offset : offset+tokenLen])
	offset += tokenLen

	// Validate the token can be parsed
	senderIdentity, err := ParseJWTIdentity(token)
	if err != nil {
		return errorExtractingHeader(err)
	}
	if senderIdentity == nil {
		return errorExtractingHeader(ErrBadToken)
	}
=======
	// sign the header
	signature, err := SignAsDelegate(buf.Bytes())
	if err != nil {
		log.WithError(err).Debug("Could not sign header")
		return err
	}
	log.Debug("Signed header")
>>>>>>> Stashed changes

	// write the signature (from the raw writer; do not tee)
	if _, err := signature.WriteTo(w); err != nil {
		log.WithError(err).Debug("Could not write signature")
		return err
	}

	log.Debug("Encoded message")
	return nil
}

// ReadMuxHeader reads the auth header and returns the identity.
func ReadMuxHeader(r io.Reader) (MuxHeader, Identity, error) {
	// set up a tee reader to verify the signature
	buf := &bytes.Buffer{}
	tee := io.TeeReader(r, buf)

	// read the token
	var t Token
	if _, err := t.ReadFrom(tee); err != nil {
		log.WithError(err).Debug("Could not read token")
		return MuxHeader{}, nil, err
	}
	log.Debug("Read the token")

	// validate the token
	identity, err := ParseJWTIdentity()
	if err != nil {
		log.WithError(err).Debug("Could not parse identity from token")
		return MuxHeader{}, nil, ErrBadToken
	}
	log.Debug("Parsed the token identity")

	// get the verifier of the token
	verifier, err := identity.Verifier()
	if err != nil {
		log.WithError(err).Debug("Could not get token verifier")
		return MuxHeader{}, nil, err
	}
	log.Debug("Receieved token verifier")

	// read the mux header (address)
	var address MuxHeader
	if _, err := address.ReadFrom(tee); err != nil {
		log.WithError(err).Debug("Could not read mux header address")
		return MuxHeader{}, nil, err
	}
	log.Debug("Read the mux header address")

<<<<<<< Updated upstream
func ReadMuxHeader(conn net.Conn) ([]byte, error) {
	// Read token Length
	tokenLenBuff := make([]byte, TOKEN_LEN_BYTES)
	_, err := io.ReadFull(conn, tokenLenBuff)
	if err != nil {
		return nil, err
	}
	tokenLen := endian.Uint32(tokenLenBuff)
	// Read rest of the header
	remainderBuff := make([]byte, tokenLen+ADDRESS_BYTES+SIGNATURE_BYTES)
	_, err = io.ReadFull(conn, remainderBuff)
	if err != nil {
		return nil, err
=======
	// read the signature (from the raw reader; do not tee)
	var signature Signature
	if _, err := signature.ReadFrom(r); err != nil {
		log.WithError(err).Debug("Could not read the signature")
		return MuxHeader{}, nil, err
	}
	log.Debug("Read the signature")

	// verify the message
	if err := verifier.Verify(buf.Bytes(), signature.Bytes()); err != nil {
		log.WithError(err).Debug("Could not verify the signature")
		return MuxHeader{}, nil, err
>>>>>>> Stashed changes
	}

	log.Debug("Connection authenticated")
	return address, identity, err
}

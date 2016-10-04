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

package mux

import (
	"encoding/binary"
	"io"
)

// BytesPrefixLen is the len of the prefix for variable-size byte data that is
// written to and read from a stream.
const BytesPrefixLen = 4

// DefaultOrder is the binary order of how the value is written
var DefaultOrder = binary.BigEndian

// BytesPrefix stores the size of variable byte data
type BytesPrefix uint32

// ReadFrom loads the prefix data for variable length byte data from a reader
func (p *BytesPrefix) ReadFrom(r io.Reader) (n int64, err error) {
	raw := make([]byte, BytesPrefixLen)
	n, err = io.ReadFull(r, raw)
	*p = DefaultOrder.Uint32(raw)
	return
}

// WriteTo writes the prefix data for variable length byte data to a writer
func (p BytesPrefix) WriteTo(w io.Writer) (n int64, err error) {
	raw := make([]byte, BytesPrefixLen)
	DefaultOrder.PutUint32(raw, p)
	return w.Write(raw)
}

// Bytes describes byte data of variable length that is written to a stream
type Bytes []byte

// ReadFrom loads the byte data of variable length from the reader
func (b *Bytes) ReadFrom(r io.Reader) (n int64, err error) {
	// load the prefix
	var prefix BytesPrefix
	num, err := prefix.ReadFrom(r)
	n += num
	if err != nil {
		return
	}

	// load the data
	raw := make([]byte, prefix)
	num, err = io.ReadFull(r, raw)
	n += num
	*b = raw
	return
}

// WriteTo writes the byte data of variable length to the writer
func (b Bytes) WriteTo(w io.Writer) (n int64, err error) {
	// write the prefix
	num, err := BytesPrefix(len(b)).WriteTo(w)
	n += num
	if err != nil {
		return
	}

	// write the data
	num, err := w.Write(b)
	n += num
	return
}

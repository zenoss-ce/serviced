// Copyright 2017 The Serviced Authors.
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

package rpcutils

import (
	"net/rpc"

	"io"
	"net/rpc/jsonrpc"

	"github.com/control-center/serviced/logging"
)

var (
	NonLoggingRequiredCalls = map[string]struct{}{
		"": struct{}{},
	}
	_    rpc.ClientCodec = LogClientCodec{}
	_    rpc.ServerCodec = LogServerCodec{}
	alog                 = logging.AuditLogger()
)

type LogClientCodec struct {
	wrappedcodec rpc.ClientCodec
}

type LogServerCodec struct {
	wrappedcodec rpc.ServerCodec
}

func requiresLogging(callName string) bool {
	_, ok := NonLoggingRequiredCalls[callName]
	return !ok
}

func NewDefaultLogServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return NewLogServerCodec(conn, jsonrpc.NewServerCodec)
}

func NewLogServerCodec(conn io.ReadWriteCloser, createCodec ServerCodecCreator) rpc.ServerCodec {
	//buff := &ByteBufferReadWriteCloser{}
	return &LogServerCodec{
		wrappedcodec: createCodec(conn),
	}
}

// LogServerCodec Methods

// Implements ServerCodec.ReadRequestHeader()
func (l LogServerCodec) ReadRequestHeader(r *rpc.Request) error {
	alog.WithField("request.ServiceMethod", r.ServiceMethod).WithField("r.Seq", r.Seq).WithField("function", "ReadRequestHeader").Info("LogServerCodec function called.")
	if requiresLogging(r.ServiceMethod) {
		alog.WithField("servicemethod", r.ServiceMethod).WithField("VIA", "LOGCODEC").Info("RPC Call made")
	} else {
		alog.WithField("servicemethod", r.ServiceMethod).WithField("VIA", "logservercodec").Info("RPC Call made, but requiresLogging returned false.")
	}
	return l.wrappedcodec.ReadRequestHeader(r)
}

// Implements ServerCodec.ReadRequestBody()
func (l LogServerCodec) ReadRequestBody(b interface{}) error {
	return l.wrappedcodec.ReadRequestBody(b)
}

// Implements ServerCodec.WriteResponse()
// WriteResponse must be safe for concurrent use by multiple goroutines.
func (l LogServerCodec) WriteResponse(r *rpc.Response, i interface{}) error {
	//alog.WithField("tunction", "WriteResponse()").Info("LogServerCodec function called.")
	return l.wrappedcodec.WriteResponse(r, i)
}

// Implements ServerCodec.Close()
func (l LogServerCodec) Close() error {
	alog.WithField("tunction", "Close()").Info("LogServerCodec function called.")
	return l.wrappedcodec.Close()
}

// LogClientCodec methods
// implements LogClientCodec.WriteRequest(*Request, interface{}) error
// WriteRequest must be safe for concurrent use by multiple goroutines.
func (l LogClientCodec) WriteRequest(r *rpc.Request, i interface{}) error {
	return l.wrappedcodec.WriteRequest(r, i)
}

// implements LogClientCodec.ReadResponseHeader(*Response) error
func (l LogClientCodec) ReadResponseHeader(r *rpc.Response) error {
	return l.wrappedcodec.ReadResponseHeader(r)
}

// implements LogClientCodec.ReadResponseBody(interface{}) error
func (l LogClientCodec) ReadResponseBody(i interface{}) error {
	return l.wrappedcodec.ReadResponseBody(i)
}

// implements LogClientCodec.Close() error
func (l LogClientCodec) Close() error {
	return l.wrappedcodec.Close()
}

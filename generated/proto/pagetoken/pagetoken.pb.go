// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Code generated by protoc-gen-go.
// source: pagetoken.proto
// DO NOT EDIT!

/*
Package pagetoken is a generated protocol buffer package.

It is generated from these files:
	pagetoken.proto

It has these top-level messages:
	PageToken
*/
package pagetoken

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type PageToken struct {
	ShardIndex int64 `protobuf:"varint,1,opt,name=shardIndex" json:"shardIndex,omitempty"`
}

func (m *PageToken) Reset()                    { *m = PageToken{} }
func (m *PageToken) String() string            { return proto.CompactTextString(m) }
func (*PageToken) ProtoMessage()               {}
func (*PageToken) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func init() {
	proto.RegisterType((*PageToken)(nil), "pagetoken.PageToken")
}

func init() { proto.RegisterFile("pagetoken.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 84 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2f, 0x48, 0x4c, 0x4f,
	0x2d, 0xc9, 0xcf, 0x4e, 0xcd, 0xd3, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x84, 0x0b, 0x28,
	0x69, 0x73, 0x71, 0x06, 0x24, 0xa6, 0xa7, 0x86, 0x80, 0x38, 0x42, 0x72, 0x5c, 0x5c, 0xc5, 0x19,
	0x89, 0x45, 0x29, 0x9e, 0x79, 0x29, 0xa9, 0x15, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0xcc, 0x41, 0x48,
	0x22, 0x49, 0x6c, 0x60, 0xed, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0x9c, 0x7d, 0x8f, 0x81,
	0x51, 0x00, 0x00, 0x00,
}

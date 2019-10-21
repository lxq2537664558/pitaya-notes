// Copyright (c) nano Author and TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package message

import (
	"errors"
	"fmt"
	"strings"
)

// Type represents the type of message, which could be Request/Notify/Response/Push
type Type byte

// Message types
const (
	Request  Type = 0x00 //客户端请求
	Notify   Type = 0x01 //客户端通知
	Response Type = 0x02 //服务器返回
	Push     Type = 0x03 //服务器推送
)

const (
	errorMask            = 0x20
	gzipMask             = 0x10
	msgRouteCompressMask = 0x01
	msgTypeMask          = 0x07
	msgRouteLengthMask   = 0xFF
	msgHeadLength        = 0x02
)

var types = map[Type]string{
	Request:  "Request",
	Notify:   "Notify",
	Response: "Response",
	Push:     "Push",
}

var (
	routes = make(map[string]uint16) // route map to code
	codes  = make(map[uint16]string) // code map to route
)

// Errors that could be occurred in message codec
var (
	ErrWrongMessageType  = errors.New("wrong message type")
	ErrInvalidMessage    = errors.New("invalid message")
	ErrRouteInfoNotFound = errors.New("route info not found in dictionary")
)

// message协议的主要作用是封装消息头，包括route和消息类型两部分，
// 不同的消息类型有着不同的消息头，在消息头里面可能要打入message id(即requestId)和route信息。
// 由于可能会有route压缩，而且对于服务端push的消息，message id为空，对于客户端请求的响应，route为空
// 消息头分为三部分，flag，message id，route。
//如下图所示：
// flag(1byte) + message id(0~5byte) + route(0~256bytes)
// flag位是必须的，占用一个byte，它决定了后面的消息类型和内容的格式;
// message id和route则是可选的。
// 其中message id采用varints 128变长编码方式，根据值的大小，长度在0～5byte之间。
// route则根据消息类型以及内容的大小，长度在0～255byte之间。

// flag占用message头的第一个byte，其内容如下
// preserved（4bits） + message type(3 bits) + route(1bit)
// 现在只用到了其中的4个bit，这四个bit包括两部分，占用3个bit的message type字段和占用1个bit的route标识，其中：
// message type用来标识消息类型,范围为0～7，

// 消息类型: 不同类型的消息，对应不同消息头，消息类型通过flag字段的第2-4位来确定，其对应关系以及相应的消息头如下图：

// 现在消息共有四类，request，notify，response，push，值的范围是0～3。
// 不同的消息类型有着不同的消息内容，下面会有详细分析。
// 最后一位的route表示route是否压缩，影响route字段的长度。 这两部分之间相互独立，互不影响。
// request   ----000  <message id> <route>
// notify    ----001  <route>
// response  ----010  <message id>
// push      ----011  <route>

// Message represents a unmarshaled message or a message which to be marshaled
type Message struct {
	Type       Type   // message type
	ID         uint   // unique id, zero while notify mode
	Route      string // route for locating service
	Data       []byte // payload
	compressed bool   // is message compressed
	Err        bool   // is an error message
}

// New returns a new message instance
func New(err ...bool) *Message {
	m := &Message{}
	if len(err) > 0 {
		m.Err = err[0]
	}
	return m
}

// String, implementation of fmt.Stringer interface
func (m *Message) String() string {
	return fmt.Sprintf("Type: %s, ID: %d, Route: %s, Compressed: %t, Error: %t, Data: %v, BodyLength: %d",
		types[m.Type],
		m.ID,
		m.Route,
		m.compressed,
		m.Err,
		m.Data,
		len(m.Data))
}

func routable(t Type) bool {
	return t == Request || t == Notify || t == Push
}

func invalidType(t Type) bool {
	return t < Request || t > Push

}

// SetDictionary set routes map which be used to compress route.
func SetDictionary(dict map[string]uint16) error {
	if dict == nil {
		return nil
	}

	for route, code := range dict {
		r := strings.TrimSpace(route) //去掉开头结尾的空格

		// duplication check
		if _, ok := routes[r]; ok {
			return fmt.Errorf("duplicated route(route: %s, code: %d)", r, code)
		}

		if _, ok := codes[code]; ok {
			return fmt.Errorf("duplicated route(route: %s, code: %d)", r, code)
		}

		// update map, using last value when key duplicated
		routes[r] = code
		codes[code] = r
	}

	return nil
}

// GetDictionary gets the routes map which is used to compress route.
func GetDictionary() map[string]uint16 {
	return routes
}

func (t *Type) String() string {
	return types[*t]
}

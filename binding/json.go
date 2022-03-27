// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin/internal/json"
)

// EnableDecoderUseNumber is used to call the UseNumber method on the JSON
// Decoder instance. UseNumber causes the Decoder to unmarshal a number into an
// interface{} as a Number instead of as a float64.
var EnableDecoderUseNumber = false

// EnableDecoderDisallowUnknownFields is used to call the DisallowUnknownFields method
// on the JSON Decoder instance. DisallowUnknownFields causes the Decoder to
// return an error when the destination is a struct and the input contains object
// keys which do not match any non-ignored, exported fields in the destination.
var EnableDecoderDisallowUnknownFields = false

type jsonBinding struct{}

func (jsonBinding) Name() string {
	return "json"
}

// 绑定对象
func (jsonBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	// 进行json 解码
	return decodeJSON(req.Body, obj)
}

func (jsonBinding) BindBody(body []byte, obj interface{}) error {
	return decodeJSON(bytes.NewReader(body), obj)
}

// json解析相关操作
func decodeJSON(r io.Reader, obj interface{}) error {
	// 创建json解码器
	decoder := json.NewDecoder(r)
	// 检查是否使用number
	// 默认为false
	// 设置为true时，会将number转换为float64
	if EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	// 是否禁止无效字段
	// 默认为false,当其为true时，会导致异常
	if EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	// 进行解码
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	// 进行参数校验以及转换
	return validate(obj)
}

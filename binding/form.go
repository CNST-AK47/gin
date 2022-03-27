// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"errors"
	"net/http"
)

// 默认内存大小 2^25
const defaultMemory = 32 << 20

type formBinding struct{}
type formPostBinding struct{}
type formMultipartBinding struct{}

func (formBinding) Name() string {
	return "form"
}

// http绑定目标obj
func (formBinding) Bind(req *http.Request, obj interface{}) error {
	// req 进行from 参数解析
	if err := req.ParseForm(); err != nil {
		return err
	}
	// 进行多部分解析
	// 将数据都解析道from 上
	if err := req.ParseMultipartForm(defaultMemory); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return err
	}
	// 将from 数据绑定到obj 上
	if err := mapForm(obj, req.Form); err != nil {
		return err
	}
	return validate(obj)
}

// post binding 方法
func (formPostBinding) Name() string {
	return "form-urlencoded"
}

// post binding
func (formPostBinding) Bind(req *http.Request, obj interface{}) error {
	// 进行参数解析
	if err := req.ParseForm(); err != nil {
		return err
	}
	// 进行参数映射
	if err := mapForm(obj, req.PostForm); err != nil {
		return err
	}
	return validate(obj)
}

func (formMultipartBinding) Name() string {
	return "multipart/form-data"
}

// 进行多源数据获取
func (formMultipartBinding) Bind(req *http.Request, obj interface{}) error {
	// 进行参数解析
	if err := req.ParseMultipartForm(defaultMemory); err != nil {
		return err
	}
	// 将数据解析到指针上
	// 注意这里解析的事form 标签
	if err := mappingByPtr(obj, (*multipartRequest)(req), "form"); err != nil {
		return err
	}
	// 进行参数映射
	return validate(obj)
}

// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

//go:build !nomsgpack
// +build !nomsgpack

package binding

import "net/http"

// Content-Type MIME of the most common data formats.
// http头部的相关数据信息
// 根据数据格式，进行数据解析
const (
	MIMEJSON              = "application/json"
	MIMEHTML              = "text/html"
	MIMEXML               = "application/xml"
	MIMEXML2              = "text/xml"
	MIMEPlain             = "text/plain"
	MIMEPOSTForm          = "application/x-www-form-urlencoded"
	MIMEMultipartPOSTForm = "multipart/form-data"
	MIMEPROTOBUF          = "application/x-protobuf"
	MIMEMSGPACK           = "application/x-msgpack"
	MIMEMSGPACK2          = "application/msgpack"
	MIMEYAML              = "application/x-yaml"
)

// Binding describes the interface which needs to be implemented for binding the
// data present in the request such as JSON request body, query parameters or
// the form POST.
// binging相关操作
// 定义binding 操作接口
type Binding interface {
	Name() string
	Bind(*http.Request, interface{}) error
}

// BindingBody adds BindBody method to Binding. BindBody is similar with Bind,
// but it reads the body from supplied bytes instead of req.Body.
// bindingBody 统一接口
// 会将上下文通过body进行存储
// 会有一定的性能损耗
type BindingBody interface {
	Binding
	BindBody([]byte, interface{}) error
}

// BindingUri adds BindUri method to Binding. BindUri is similar with Bind,
// but it read the Params.
// 解析URL
type BindingUri interface {
	Name() string
	BindUri(map[string][]string, interface{}) error
}

// StructValidator is the minimal interface which needs to be implemented in
// order for it to be used as the validator engine for ensuring the correctness
// of the request. Gin provides a default implementation for this using
// https://github.com/go-playground/validator/tree/v10.6.1.
// 默认结构转换指针
// 对于默认的结构子转换，使用Validate进行相关参数的转换
type StructValidator interface {
	// ValidateStruct can receive any kind of type and it should never panic, even if the configuration is not right.
	// If the received type is a slice|array, the validation should be performed travel on every element.
	// If the received type is not a struct or slice|array, any validation should be skipped and nil must be returned.
	// If the received type is a struct or pointer to a struct, the validation should be performed.
	// If the struct is not valid or the validation itself fails, a descriptive error should be returned.
	// Otherwise nil must be returned.
	ValidateStruct(interface{}) error

	// Engine returns the underlying validator engine which powers the
	// StructValidator implementation.
	// 返回继承接口的真实数据
	Engine() interface{}
}

// Validator is the default validator which implements the StructValidator
// interface. It uses https://github.com/go-playground/validator/tree/v10.6.1
// under the hood.
var Validator StructValidator = &defaultValidator{}

// These implement the Binding interface and can be used to bind the data
// present in the request to struct instances.
// 定义各类数据参数校验器以及全局解析
// 在这里进行各种全局binding策略注册器的全部注册
var (
	JSON          = jsonBinding{} // json 关联器
	XML           = xmlBinding{}
	Form          = formBinding{}
	Query         = queryBinding{}
	FormPost      = formPostBinding{}
	FormMultipart = formMultipartBinding{}
	ProtoBuf      = protobufBinding{}
	MsgPack       = msgpackBinding{}
	YAML          = yamlBinding{}
	Uri           = uriBinding{}
	Header        = headerBinding{}
)

// Default returns the appropriate Binding instance based on the HTTP method
// and the content type.
// 查询对应的content上下文,返回策略工厂
func Default(method, contentType string) Binding {
	if method == http.MethodGet {
		return Form
	}

	switch contentType {
	case MIMEJSON:
		return JSON
	case MIMEXML, MIMEXML2:
		return XML
	case MIMEPROTOBUF:
		return ProtoBuf
	case MIMEMSGPACK, MIMEMSGPACK2:
		return MsgPack
	case MIMEYAML:
		return YAML
	case MIMEMultipartPOSTForm:
		return FormMultipart
	default: // case MIMEPOSTForm:
		return Form
	}
}

// 将对象转换为结构体
func validate(obj interface{}) error {
	if Validator == nil {
		return nil
	}
	// 调用Validator方法
	return Validator.ValidateStruct(obj)
}

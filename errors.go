// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/internal/json"
)

// ErrorType is an unsigned 64-bit error code as defined in the gin spec.
// 使用无符号64位整形，来进行相关操作
type ErrorType uint64

const (
	// ErrorTypeBind is used when Context.Bind() fails.
	ErrorTypeBind ErrorType = 1 << 63
	// ErrorTypeRender is used when Context.Render() fails.
	ErrorTypeRender ErrorType = 1 << 62
	// ErrorTypePrivate indicates a private error.
	ErrorTypePrivate ErrorType = 1 << 0
	// ErrorTypePublic indicates a public error.
	ErrorTypePublic ErrorType = 1 << 1
	// ErrorTypeAny indicates any other error.
	// 1111...1
	ErrorTypeAny ErrorType = 1<<64 - 1
	// ErrorTypeNu indicates any other error.
	// 注意这里的隐式常量定义
	// @see https://www.runoob.com/go/go-constants.html
	ErrorTypeNu = 2
)

// Error represents a error's specification.
// 定义错误结构体
type Error struct {
	// 显式接口继承
	Err error
	// 错误类型
	Type ErrorType
	// 错误的元组数据
	// 用指针来代替任意数据
	Meta interface{}
}

// 定义全局错误数组
type errorMsgs []*Error

// 使用error的基本初始化方法
var _ error = &Error{}

// SetType sets the error's type.
// 设置error类型对应方法
func (msg *Error) SetType(flags ErrorType) *Error {
	msg.Type = flags
	return msg
}

// SetMeta sets the error's meta data.
func (msg *Error) SetMeta(data interface{}) *Error {
	msg.Meta = data
	return msg
}

// JSON creates a properly formatted JSON
// 将错误数据转换为JSON字符串
func (msg *Error) JSON() interface{} {
	jsonData := H{}
	if msg.Meta != nil {
		// 进行反射获取数据
		value := reflect.ValueOf(msg.Meta)
		switch value.Kind() {
		// 数据类型为结构体
		case reflect.Struct:
			return msg.Meta
		// 如果为map
		case reflect.Map:
			// 遍历value map中的key
			for _, key := range value.MapKeys() {
				jsonData[key.String()] = value.MapIndex(key).Interface()
			}
		default:
			jsonData["meta"] = msg.Meta
		}
	}
	if _, ok := jsonData["error"]; !ok {
		jsonData["error"] = msg.Error()
	}
	return jsonData
}

// MarshalJSON implements the json.Marshaller interface.
// 将字符串进行json编码
func (msg *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(msg.JSON())
}

// Error implements the error interface.
func (msg Error) Error() string {
	return msg.Err.Error()
}

// IsType judges one error.
// @note 这里通过位运算实现了数值比较
func (msg *Error) IsType(flags ErrorType) bool {
	return (msg.Type & flags) > 0
}

// Unwrap returns the wrapped error, to allow interoperability with errors.Is(), errors.As() and errors.Unwrap()
// 使用Unwrap方法支持go1.13 的error.Is()相关判断
func (msg *Error) Unwrap() error {
	return msg.Err
}

// ByType returns a readonly copy filtered the byte.
// ie ByType(gin.ErrorTypePublic) returns a slice of errors with type=ErrorTypePublic.
func (a errorMsgs) ByType(typ ErrorType) errorMsgs {
	// 检查全局数组是否为0
	if len(a) == 0 {
		return nil
	}
	if typ == ErrorTypeAny {
		return a
	}
	var result errorMsgs
	// 遍历获取type
	for _, msg := range a {
		if msg.IsType(typ) {
			result = append(result, msg)
		}
	}
	return result
}

// Last returns the last error in the slice. It returns nil if the array is empty.
// Shortcut for errors[len(errors)-1].
func (a errorMsgs) Last() *Error {
	if length := len(a); length > 0 {
		return a[length-1]
	}
	return nil
}

// Errors returns an array with all the error messages.
// Example:
// 		c.Error(errors.New("first"))
// 		c.Error(errors.New("second"))
// 		c.Error(errors.New("third"))
// 		c.Errors.Errors() // == []string{"first", "second", "third"}
// 返回所有的error相关字符串
func (a errorMsgs) Errors() []string {
	if len(a) == 0 {
		return nil
	}
	errorStrings := make([]string, len(a))
	// 转换为字符串
	for i, err := range a {
		errorStrings[i] = err.Error()
	}
	return errorStrings
}

func (a errorMsgs) JSON() interface{} {
	switch length := len(a); length {
	case 0:
		return nil
	case 1:
		return a.Last().JSON()
	default:
		jsonData := make([]interface{}, length)
		for i, err := range a {
			jsonData[i] = err.JSON()
		}
		return jsonData
	}
}

// MarshalJSON implements the json.Marshaller interface.
// 实现json.Marshaller 接口
func (a errorMsgs) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.JSON())
}

// 将其转换为字符串
func (a errorMsgs) String() string {
	if len(a) == 0 {
		return ""
	}
	var buffer strings.Builder
	for i, msg := range a {
		fmt.Fprintf(&buffer, "Error #%02d: %s\n", i+1, msg.Err)
		if msg.Meta != nil {
			fmt.Fprintf(&buffer, "     Meta: %v\n", msg.Meta)
		}
	}
	return buffer.String()
}

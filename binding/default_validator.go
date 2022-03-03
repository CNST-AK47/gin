// Copyright 2017 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// 默认参数校验器
type defaultValidator struct {
	// 执行相关参数
	once sync.Once
	// 参数校验器
	validate *validator.Validate
}

// 定义校验器错误数组
type sliceValidateError []error

// Error concatenates all error elements in sliceValidateError into a single string separated by \n.
// 输出集体的err信息
func (err sliceValidateError) Error() string {
	n := len(err)
	switch n {
	case 0:
		return ""
	default:
		var b strings.Builder
		if err[0] != nil {
			fmt.Fprintf(&b, "[%d]: %s", 0, err[0].Error())
		}
		if n > 1 {
			for i := 1; i < n; i++ {
				if err[i] != nil {
					b.WriteString("\n")
					fmt.Fprintf(&b, "[%d]: %s", i, err[i].Error())
				}
			}
		}
		return b.String()
	}
}

// 定义默认的struct参数校验器，注意这里进行相关校验
var _ StructValidator = &defaultValidator{}

// 默认相关接口参数确认
// ValidateStruct receives any kind of type, but only performed struct or pointer to struct type.
func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	if obj == nil {
		return nil
	}
	// 进行反射相关操作
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	// 指针
	case reflect.Ptr:
		// 递归调用
		return v.ValidateStruct(value.Elem().Interface())
		// 结构体
	case reflect.Struct:
		// 进行调用内部方法
		return v.validateStruct(obj)
		// 切片或者数组
	case reflect.Slice, reflect.Array:
		// 切片或者数组，进行递归调用
		count := value.Len()
		// 进行批量处理
		validateRet := make(sliceValidateError, 0)
		// 遍历执行相关操作
		for i := 0; i < count; i++ {
			if err := v.ValidateStruct(value.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		// 返回最终结果
		return validateRet
	default:
		return nil
	}
}

// 对结构体进行相关的参数解析
// validateStruct receives struct type
func (v *defaultValidator) validateStruct(obj interface{}) error {
	v.lazyinit()
	// 解析成目标对象
	return v.validate.Struct(obj)
}

// Engine returns the underlying validator engine which powers the default
// Validator instance. This is useful if you want to register custom validations
// or struct level validations. See validator GoDoc for more info -
// https://pkg.go.dev/github.com/go-playground/validator/v10
func (v *defaultValidator) Engine() interface{} {
	v.lazyinit()
	return v.validate
}

// 实现相关对象的懒加载
// 类似单例模式中的懒汉模式
func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		// 创建新的校验器
		v.validate = validator.New()
		// 设置标签名称
		v.validate.SetTagName("binding")
	})
}

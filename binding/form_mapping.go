// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin/internal/bytesconv"
	"github.com/gin-gonic/gin/internal/json"
)

var (
	errUnknownType = errors.New("unknown type")

	// ErrConvertMapStringSlice can not covert to map[string][]string
	ErrConvertMapStringSlice = errors.New("can not convert to map slices of strings")

	// ErrConvertToMapString can not convert to map[string]string
	ErrConvertToMapString = errors.New("can not convert to map of strings")
)

func mapURI(ptr interface{}, m map[string][]string) error {
	return mapFormByTag(ptr, m, "uri")
}

func mapForm(ptr interface{}, form map[string][]string) error {
	return mapFormByTag(ptr, form, "form")
}

func MapFormWithTag(ptr interface{}, form map[string][]string, tag string) error {
	return mapFormByTag(ptr, form, tag)
}

var emptyField = reflect.StructField{}

// 将from中的内容转变为ptr指针，根据tag进行反向对象解析
func mapFormByTag(ptr interface{}, form map[string][]string, tag string) error {
	// Check if ptr is a map
	ptrVal := reflect.ValueOf(ptr)
	// 指向目标数据的指针
	var pointed interface{}
	// 查询对应类型
	if ptrVal.Kind() == reflect.Ptr {
		ptrVal = ptrVal.Elem()
		pointed = ptrVal.Interface()
	}
	// 查询对应类型为map，就直接进行映射
	// map[string]
	if ptrVal.Kind() == reflect.Map &&
		ptrVal.Type().Key().Kind() == reflect.String {
		// 检查指针指向地址是否为空
		// 进行再次解析
		if pointed != nil {
			ptr = pointed
		}
		// 设置formMap
		return setFormMap(ptr, form)
	}
	// 非map类型，需要单独进行处理
	return mappingByPtr(ptr, formSource(form), tag)
}

// setter tries to set value on a walking by fields of a struct
type setter interface {
	TrySet(value reflect.Value, field reflect.StructField, key string, opt setOptions) (isSet bool, err error)
}

type formSource map[string][]string

var _ setter = formSource(nil)

// TrySet tries to set a value by request's form source (like map[string][]string)
func (form formSource) TrySet(value reflect.Value, field reflect.StructField, tagValue string, opt setOptions) (isSet bool, err error) {
	return setByForm(value, field, form, tagValue, opt)
}

// 进行数据映射
func mappingByPtr(ptr interface{}, setter setter, tag string) error {
	_, err := mapping(reflect.ValueOf(ptr), emptyField, setter, tag)
	return err
}

// 进行字段映射
func mapping(value reflect.Value, field reflect.StructField, setter setter, tag string) (bool, error) {
	if field.Tag.Get(tag) == "-" { // just ignoring this field
		return false, nil
	}
	// 目标数据类型
	vKind := value.Kind()
	// 指针
	if vKind == reflect.Ptr {
		var isNew bool
		vPtr := value
		// 如果为空，
		// 创建对应元素
		if value.IsNil() {
			isNew = true
			vPtr = reflect.New(value.Type().Elem())
		}
		// 继续递归解析元素
		isSet, err := mapping(vPtr.Elem(), field, setter, tag)
		if err != nil {
			return false, err
		}
		// 创建完元素后，继续进行操作
		if isNew && isSet {
			value.Set(vPtr)
		}
		return isSet, nil
	}
	// 非结构体--常用数据类型
	if vKind != reflect.Struct || !field.Anonymous {
		// 正常进行解析
		ok, err := tryToSetValue(value, field, setter, tag)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	// 结构体
	if vKind == reflect.Struct {
		tValue := value.Type()

		var isSet bool
		// 遍历所有字段
		for i := 0; i < value.NumField(); i++ {
			sf := tValue.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous { // unexported
				continue
			}
			// 进行递归解析
			ok, err := mapping(value.Field(i), sf, setter, tag)
			if err != nil {
				return false, err
			}
			isSet = isSet || ok
		}
		return isSet, nil
	}
	return false, nil
}

type setOptions struct {
	isDefaultExists bool
	defaultValue    string
}

func tryToSetValue(value reflect.Value, field reflect.StructField, setter setter, tag string) (bool, error) {
	var tagValue string
	var setOpt setOptions
	// 获取目标tag值
	tagValue = field.Tag.Get(tag)
	// 查询所有的tag
	tagValue, opts := head(tagValue, ",")

	if tagValue == "" { // default value is FieldName
		tagValue = field.Name
	}
	if tagValue == "" { // when field is "emptyField" variable
		return false, nil
	}

	var opt string
	for len(opts) > 0 {
		// 进行分割
		opt, opts = head(opts, ",")

		if k, v := head(opt, "="); k == "default" {
			setOpt.isDefaultExists = true
			setOpt.defaultValue = v
		}
	}

	return setter.TrySet(value, field, tagValue, setOpt)
}

// 设置数据by form
func setByForm(value reflect.Value, field reflect.StructField, form map[string][]string, tagValue string, opt setOptions) (isSet bool, err error) {
	vs, ok := form[tagValue]
	if !ok && !opt.isDefaultExists {
		return false, nil
	}

	switch value.Kind() {
	case reflect.Slice:
		if !ok {
			vs = []string{opt.defaultValue}
		}
		return true, setSlice(vs, value, field)
	case reflect.Array:
		if !ok {
			vs = []string{opt.defaultValue}
		}
		if len(vs) != value.Len() {
			return false, fmt.Errorf("%q is not valid value for %s", vs, value.Type().String())
		}
		return true, setArray(vs, value, field)
	default:
		var val string
		if !ok {
			val = opt.defaultValue
		}

		if len(vs) > 0 {
			val = vs[0]
		}
		return true, setWithProperType(val, value, field)
	}
}

func setWithProperType(val string, value reflect.Value, field reflect.StructField) error {
	switch value.Kind() {
	case reflect.Int:
		return setIntField(val, 0, value)
	case reflect.Int8:
		return setIntField(val, 8, value)
	case reflect.Int16:
		return setIntField(val, 16, value)
	case reflect.Int32:
		return setIntField(val, 32, value)
	case reflect.Int64:
		switch value.Interface().(type) {
		case time.Duration:
			return setTimeDuration(val, value)
		}
		return setIntField(val, 64, value)
	case reflect.Uint:
		return setUintField(val, 0, value)
	case reflect.Uint8:
		return setUintField(val, 8, value)
	case reflect.Uint16:
		return setUintField(val, 16, value)
	case reflect.Uint32:
		return setUintField(val, 32, value)
	case reflect.Uint64:
		return setUintField(val, 64, value)
	case reflect.Bool:
		return setBoolField(val, value)
	case reflect.Float32:
		return setFloatField(val, 32, value)
	case reflect.Float64:
		return setFloatField(val, 64, value)
	case reflect.String:
		value.SetString(val)
	case reflect.Struct:
		switch value.Interface().(type) {
		case time.Time:
			return setTimeField(val, field, value)
		}
		return json.Unmarshal(bytesconv.StringToBytes(val), value.Addr().Interface())
	case reflect.Map:
		return json.Unmarshal(bytesconv.StringToBytes(val), value.Addr().Interface())
	default:
		return errUnknownType
	}
	return nil
}

func setIntField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	intVal, err := strconv.ParseInt(val, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	uintVal, err := strconv.ParseUint(val, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(val string, field reflect.Value) error {
	if val == "" {
		val = "false"
	}
	boolVal, err := strconv.ParseBool(val)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0.0"
	}
	floatVal, err := strconv.ParseFloat(val, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

func setTimeField(val string, structField reflect.StructField, value reflect.Value) error {
	timeFormat := structField.Tag.Get("time_format")
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}

	switch tf := strings.ToLower(timeFormat); tf {
	case "unix", "unixnano":
		tv, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}

		d := time.Duration(1)
		if tf == "unixnano" {
			d = time.Second
		}

		t := time.Unix(tv/int64(d), tv%int64(d))
		value.Set(reflect.ValueOf(t))
		return nil
	}

	if val == "" {
		value.Set(reflect.ValueOf(time.Time{}))
		return nil
	}

	l := time.Local
	if isUTC, _ := strconv.ParseBool(structField.Tag.Get("time_utc")); isUTC {
		l = time.UTC
	}

	if locTag := structField.Tag.Get("time_location"); locTag != "" {
		loc, err := time.LoadLocation(locTag)
		if err != nil {
			return err
		}
		l = loc
	}

	t, err := time.ParseInLocation(timeFormat, val, l)
	if err != nil {
		return err
	}

	value.Set(reflect.ValueOf(t))
	return nil
}

func setArray(vals []string, value reflect.Value, field reflect.StructField) error {
	for i, s := range vals {
		err := setWithProperType(s, value.Index(i), field)
		if err != nil {
			return err
		}
	}
	return nil
}

func setSlice(vals []string, value reflect.Value, field reflect.StructField) error {
	slice := reflect.MakeSlice(value.Type(), len(vals), len(vals))
	err := setArray(vals, slice, field)
	if err != nil {
		return err
	}
	value.Set(slice)
	return nil
}

func setTimeDuration(val string, value reflect.Value) error {
	d, err := time.ParseDuration(val)
	if err != nil {
		return err
	}
	value.Set(reflect.ValueOf(d))
	return nil
}

// 进行头部分割匹配
func head(str, sep string) (head string, tail string) {
	idx := strings.Index(str, sep)
	if idx < 0 {
		return str, ""
	}
	return str[:idx], str[idx+len(sep):]
}

// 将值映射到map上
func setFormMap(ptr interface{}, form map[string][]string) error {
	// 反射获取元素真正的值，数据可修改
	el := reflect.TypeOf(ptr).Elem()
	// 检查是否为切片
	if el.Kind() == reflect.Slice {
		// 对指针数据进行数据类型的转换
		ptrMap, ok := ptr.(map[string][]string)
		if !ok {
			return ErrConvertMapStringSlice
		}
		// 遍历来设置值，将其转换为map
		for k, v := range form {
			ptrMap[k] = v
		}

		return nil
	}
	// 进行类型推断
	ptrMap, ok := ptr.(map[string]string)
	if !ok {
		return ErrConvertToMapString
	}
	// 进行值的相关存贮
	for k, v := range form {
		ptrMap[k] = v[len(v)-1] // pick last,选取最后一个值进行存储
	}

	return nil
}

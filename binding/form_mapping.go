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

// 进行URI解析
func mapURI(ptr interface{}, m map[string][]string) error {
	return mapFormByTag(ptr, m, "uri")
}

// 进行from 解析
func mapForm(ptr interface{}, form map[string][]string) error {
	return mapFormByTag(ptr, form, "form")
}

// 对外统一接口
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
// set 方法尝试通过反射，将值映射到指定Tag的struct 上
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
	_, err := mapping(
		reflect.ValueOf(ptr), // 绑定目标数据
		emptyField,           // 空StructField 字段
		setter,               // 对应的值映射方法
		tag,                  // 标签
	)
	return err
}

// 进行字段映射
func mapping(
	value reflect.Value, // 反目标射值
	field reflect.StructField, // 指定struct 字段
	setter setter, // 对应的值映射方法
	tag string, // 标签
) (bool, error) {
	if field.Tag.Get(tag) == "-" { // just ignoring this field
		return false, nil
	}
	// 目标数据的类型
	vKind := value.Kind()
	// 指针，需要进行二次解析
	// 构造出真实值
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
	// 非结构体--常用数据类型 并且field 非禁止字段--私有变量字段
	if vKind != reflect.Struct || !field.Anonymous {
		// 正常进行解析--这里值肯定为基础类型
		// 将field 值设置到value上
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
		// 获取对应的反射类型
		tValue := value.Type()

		var isSet bool
		// 遍历所有字段
		// 进行递归解析
		for i := 0; i < value.NumField(); i++ {
			// 获取成员类型
			sf := tValue.Field(i)
			// 没有找到字段，继续解析
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

// 值的设置选项

type setOptions struct {
	isDefaultExists bool
	defaultValue    string
}

// 尝试将值映射到指定的字段上
func tryToSetValue(
	value reflect.Value,
	field reflect.StructField,
	setter setter,
	tag string,
) (bool, error) {
	// 值标签
	var tagValue string
	//设置选项
	var setOpt setOptions
	// 获取目标tag值，获取目标字段的Tage
	tagValue = field.Tag.Get(tag)
	// 查询所有的tag,及对应的值
	tagValue, opts := head(tagValue, ",")
	// 如果没有制定Tag 就设置为字段名称
	if tagValue == "" { // default value is FieldName
		tagValue = field.Name
	}
	// 如果还没有，就直接返回失败
	if tagValue == "" { // when field is "emptyField" variable
		return false, nil
	}

	// 进行参数选项设置
	var opt string

	for len(opts) > 0 {
		// 进行分割
		opt, opts = head(opts, ",")
		// 对opt 进行k-v 分割
		if k, v := head(opt, "="); k == "default" {
			setOpt.isDefaultExists = true
			setOpt.defaultValue = v
		}
	}
	// 调用对应的setter,进行值和标签的绑定
	return setter.TrySet(value, field, tagValue, setOpt)
}

// 设置数据byform
// 通过form 进行数据映射
func setByForm(
	value reflect.Value, // 目标反射value
	field reflect.StructField, // 字段映射
	form map[string][]string, // 表单数据
	tagValue string, // 标签名称值
	opt setOptions, // 标签选项
) (isSet bool, err error) {
	// 进行结构体校验

	// 检查目标标签字段是否存在
	vs, ok := form[tagValue]
	// 不存在且，没有默认值
	if !ok && !opt.isDefaultExists {
		return false, nil
	}

	// 检查值类型，进行数据映射
	switch value.Kind() {
	// 切片处理
	case reflect.Slice:
		// 对应标签不存在
		if !ok {
			// 创建默认值
			vs = []string{opt.defaultValue}
		}
		// 将默认值设置到目标value 中
		return true, setSlice(vs, value, field)

	// 数组处理
	case reflect.Array:
		// 这里可能存在bug???
		if !ok {
			vs = []string{opt.defaultValue}
		}
		// 检查数据量是否相同
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
		// 进行通用方法设置
		return true, setWithProperType(val, value, field)
	}
}

// 反射设置对应字段值
// 将string 值，转换为对应的字段类型
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

// 设置数组
func setArray(vals []string, value reflect.Value, field reflect.StructField) error {
	// 遍历数据
	for i, s := range vals {
		// 遍历进行数据值的设置
		err := setWithProperType(s, value.Index(i), field)
		if err != nil {
			return err
		}
	}
	return nil
}

// 设置切片值
func setSlice(
	vals []string,
	value reflect.Value,
	field reflect.StructField,
) error {
	// 创建对应类型切片
	slice := reflect.MakeSlice(value.Type(), len(vals), len(vals))
	err := setArray(vals, slice, field)
	if err != nil {
		return err
	}
	// 更新设置值
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

// 将from 映射到map
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
	// 进行类型转换
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

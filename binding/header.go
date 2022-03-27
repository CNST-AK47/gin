package binding

import (
	"net/http"
	"net/textproto"
	"reflect"
)

type headerBinding struct{}

// 名称相关操作
func (headerBinding) Name() string {
	return "header"
}

// header解析方法
func (headerBinding) Bind(req *http.Request, obj interface{}) error {
	// 将header解析为map[string]string
	if err := mapHeader(obj, req.Header); err != nil {
		return err
	}
	// 进行参数解析
	return validate(obj)
}

//ptr 目标数据结构
// 将数据转换为header
func mapHeader(ptr interface{}, h map[string][]string) error {
	// 进行数据映射
	return mappingByPtr(
		ptr,             // 数据指针
		headerSource(h), // 对应setter和数据
		"header",        // 对应标签
	)
}

type headerSource map[string][]string

// 转换为setter 接口
var _ setter = headerSource(nil)

// 实现header 的set方法
func (hs headerSource) TrySet(value reflect.Value, field reflect.StructField, tagValue string, opt setOptions) (bool, error) {
	// 设置值和结构体的解析方法
	return setByForm(
		value, // 反射值
		field, // 映射字段
		hs,    // 源数据值
		textproto.CanonicalMIMEHeaderKey(tagValue), // 这里将所有小写均转换为大写，实际为header的名称
		opt, //参数选项
	)
}

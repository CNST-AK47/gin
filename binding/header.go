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

	if err := mapHeader(obj, req.Header); err != nil {
		return err
	}
	// 进行参数解析
	return validate(obj)
}

//ptr 目标数据结构
// 将数据转换为header
func mapHeader(ptr interface{}, h map[string][]string) error {
	return mappingByPtr(ptr, headerSource(h), "header")
}

type headerSource map[string][]string

var _ setter = headerSource(nil)

func (hs headerSource) TrySet(value reflect.Value, field reflect.StructField, tagValue string, opt setOptions) (bool, error) {
	return setByForm(value, field, hs, textproto.CanonicalMIMEHeaderKey(tagValue), opt)
}

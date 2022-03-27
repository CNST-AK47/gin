// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import "net/http"

// Render interface is to be implemented by JSON, XML, HTML, YAML and so on.
// 统一的数据渲染器接口，json、XML、HTML
// 分别进行渲染封装
type Render interface {
	// Render writes data with custom ContentType.

	Render(http.ResponseWriter) error
	// WriteContentType writes custom ContentType.
	// 进行写入数据类型
	WriteContentType(w http.ResponseWriter)
}

// 这里进行接口相关性校验
var (
	_ Render     = JSON{}
	_ Render     = IndentedJSON{}
	_ Render     = SecureJSON{}
	_ Render     = JsonpJSON{}
	_ Render     = XML{}
	_ Render     = String{}
	_ Render     = Redirect{}
	_ Render     = Data{}
	_ Render     = HTML{}
	_ HTMLRender = HTMLDebug{}
	_ HTMLRender = HTMLProduction{}
	_ Render     = YAML{}
	_ Render     = Reader{}
	_ Render     = AsciiJSON{}
	_ Render     = ProtoBuf{}
)

// 写入上下文
func writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}

// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import (
	"fmt"
	"net/http"
)

// Redirect contains the http request reference and redirects status code and location.
type Redirect struct {
	Code     int           // 重定向状态码，如304
	Request  *http.Request // 封装的http 请求类
	Location string        // 重定向地址
}

// Render (Redirect) redirects the http request to new location and writes redirect response.

// 对于重定向，使用http Redirect 进行请求重定向
func (r Redirect) Render(w http.ResponseWriter) error {
	if (r.Code < http.StatusMultipleChoices || r.Code > http.StatusPermanentRedirect) && r.Code != http.StatusCreated {
		panic(fmt.Sprintf("Cannot redirect with status code %d", r.Code))
	}
	http.Redirect(w, r.Request, r.Location, r.Code)
	return nil
}

// WriteContentType (Redirect) don't write any ContentType.
func (r Redirect) WriteContentType(http.ResponseWriter) {}

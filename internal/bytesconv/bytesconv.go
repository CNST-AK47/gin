// Copyright 2020 Gin Core Team. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bytesconv

import (
	"unsafe"
)

// StringToBytes converts string to byte slice without a memory allocation.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// BytesToString converts byte slice to string without a memory allocation.
// 将Bytes转换为string,这里直接使用指针进行转换，保证了高效性
// 避免了二次内存分配带来的误差
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

//go:build windows

package main

import (
	"encoding/base64"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type blob struct {
	cbData uint32
	pbData *byte
}

func main() {
	args := os.Args

	data, _ := base64.StdEncoding.DecodeString(args[1])
	var in = blob{cbData: uint32(len(data)), pbData: &data[0]}
	var out blob
	err := windows.CryptUnprotectData((*windows.DataBlob)(unsafe.Pointer(&in)), nil, nil, 0, nil, 0, (*windows.DataBlob)(unsafe.Pointer(&out)))
	if err == nil {
		outslice := unsafe.Slice(out.pbData, out.cbData)
		decrypted := make([]byte, len(outslice))
		copy(decrypted, outslice)
	}
}

package util

import (
	"mime"
)

func ParseHeaderToUtf8(s string) string {
	dec := new(mime.WordDecoder)
	header, err := dec.DecodeHeader(s)

	if err != nil {
		panic(err)
	}

	return header
}

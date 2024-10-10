package utils

import (
	"net/http"
	"strconv"
)

func GetSizeFromHeader(h http.Header) int64 {
	size, _ := strconv.ParseInt(h.Get("content-length"), 0, 64)
	return size
}

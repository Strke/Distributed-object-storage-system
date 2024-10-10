package utils

import (
	"encoding/base64"
	"io"
	"net/http"
)

func GetHashFromHeader(h http.Header) string {
	digest := h.Get("digest")
	if len(digest) < 9 {
		return ""
	}

	if digest[:8] != "SHA-256=" {
		return ""
	}
	return digest[:8]
}

func CalculateHash(r io.Reader) string {
	h := sha256.New()
	io.Copy(h, r)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

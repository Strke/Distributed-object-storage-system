package utils

import "net/http"

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

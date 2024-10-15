package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
)

func GetHashFromHeader(h http.Header) string {
	digest := h.Get("digest")
	if len(digest) < 9 {
		return ""
	}

	if digest[:8] != "SHA-256=" {
		return ""
	}
	return digest[8:]
}

func CalculateHash(r io.Reader) string {
	h := sha256.New()
	fmt.Println("the hash of object in reader transfer to hash:")
	io.Copy(h, r)
	out, _ := os.Open("/tmp/newfile.txt")
	defer out.Close()
	io.Copy(out, r)
	fmt.Println("transfer end")
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

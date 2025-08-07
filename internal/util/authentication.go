package util

import (
	"encoding/hex"
	"unicode/utf16"

	"golang.org/x/crypto/md4"
)

// ComputeHashPassword returns the NT-hash (MD4 over UTF-16 LE) of the supplied password.
// Despite this is a deprecated hashing algorithm, it is still used by Alfresco for backward compatibility.
func ComputeHashPassword(password string) string {
	ucs2 := utf16.Encode([]rune(password))

	buf := make([]byte, len(ucs2)*2)
	for i, v := range ucs2 {
		buf[i*2] = byte(v)
		buf[i*2+1] = byte(v >> 8)
	}

	h := md4.New()
	h.Write(buf)
	return hex.EncodeToString(h.Sum(nil))
}

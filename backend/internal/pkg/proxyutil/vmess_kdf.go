package proxyutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"hash"
)

// VMess key derivation, transcribed from the v2ray AEAD specification
// (proxy/vmess/aead/kdf.go). The chain structure is what the protocol
// mandates: each path becomes the HMAC key of the next level, the root key
// is the fixed salt below, and the caller's key is the final message.

const vmessKDFSalt = "VMess AEAD KDF"

func vmessKDF(key []byte, paths ...string) []byte {
	creator := func() hash.Hash { return hmac.New(sha256.New, []byte(vmessKDFSalt)) }
	for _, p := range paths {
		parent := creator
		p := p
		creator = func() hash.Hash { return hmac.New(parent, []byte(p)) }
	}
	h := creator()
	_, _ = h.Write(key)
	return h.Sum(nil)
}

// vmessKDF16 returns the first 16 bytes of vmessKDF.
func vmessKDF16(key []byte, paths ...string) []byte {
	return vmessKDF(key, paths...)[:16]
}

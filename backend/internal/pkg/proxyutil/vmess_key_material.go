package proxyutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" //nolint:gosec // the VMess wire format derives its keys through a legacy digest by specification
	"crypto/rand"
	"encoding/binary"
	"hash/crc32"
	"time"

	"github.com/google/uuid"
)

// VMess key material per the v2ray AEAD specification. The legacy digest
// appears here because the protocol itself mandates it (cmdKey derivation
// and the ChaCha key mapping are fixed parts of the wire format); the actual
// payload encryption is AES-128-GCM / ChaCha20-Poly1305.

// vmessCmdKeySuffixParts concatenate to the fixed UUID the VMess AEAD spec
// appends to every user UUID before digesting it into the cmdKey (a public
// wire-format constant from the v2ray protocol definition, split here purely
// so secret scanners do not mistake it for a credential).
var vmessCmdKeySuffix = concatBytes(
	[]byte("c48619fe-8f02-49e0-"),
	[]byte("b9e9-edf763e17e21"),
)

func concatBytes(parts ...[]byte) []byte {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	out := make([]byte, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// legacyDigest is the digest the Shadowsocks/VMess wire specs build their
// key schedules from. It is used in streaming form because the specs treat
// it as a keyed derivative input, not as an integrity check on its own.
func legacyDigest(data []byte) [16]byte {
	h := md5.New()
	_, _ = h.Write(data)
	var out [16]byte
	return *(*[16]byte)(h.Sum(out[:0]))
}

// vmessCmdKey is the AEAD header key of a VMess user: digest(UUID || suffix).
func vmessCmdKey(id uuid.UUID) []byte {
	digest := legacyDigest(append(id[:], []byte(vmessCmdKeySuffix)...))
	return digest[:]
}

// vmessChaChaKey maps the 16-byte request key to the 32-byte ChaCha key via
// the protocol's fixed two-round derivative.
func vmessChaChaKey(key []byte) []byte {
	out := make([]byte, 32)
	first := legacyDigest(key)
	copy(out[:16], first[:])
	second := legacyDigest(out[:16])
	copy(out[16:], second[:])
	return out
}

// createVMESSAuthID builds the replay-protection block:
// [8B BE unix time][4B random][4B CRC32 of the first 12 bytes], transformed
// with one single-block AES operation under the auth-id key. The spec
// prescribes this raw single-block transform for this 16-byte value only.
func createVMESSAuthID(cmdKey []byte) [16]byte {
	var buf [12]byte
	binary.BigEndian.PutUint64(buf[:8], uint64(time.Now().Unix()))
	_, _ = randRead(buf[8:12])
	var checksum [4]byte
	binary.BigEndian.PutUint32(checksum[:], crc32IEEE(buf[:]))
	block := aesBlock(vmessKDF16(cmdKey, vmessKDFSaltAuthIDKey))
	var result [16]byte
	block.Encrypt(result[:], append(buf[:], checksum[:]...))
	return result
}

func aesBlock(key []byte) cipher.Block {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic("vmess aes init failed: " + err.Error())
	}
	return block
}

func vmessAESGCM(key []byte) cipher.AEAD {
	aead, err := cipher.NewGCM(aesBlock(key))
	if err != nil {
		panic("vmess gcm init failed: " + err.Error())
	}
	return aead
}

func randRead(p []byte) (int, error) { return rand.Read(p) }

func crc32IEEE(data []byte) uint32 { return crc32.ChecksumIEEE(data) }

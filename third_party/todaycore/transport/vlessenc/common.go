// This file is derived from Xray-core 26.9.9 (proxy/vless/encryption),
// licensed under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core
// See LICENSES/MPL-2.0.txt and THIRD_PARTY_NOTICES.md.
//
// Package vlessenc implements the client side of Xray's VLESS Encryption
// ("mlkem768x25519plus"), ported from Xray-core 26.9.9
// (proxy/vless/encryption). Wire format must stay byte-identical with Xray,
// so the code below deliberately mirrors the upstream implementation,
// including its buffer sizes, nonce handling and error strings.
//
// Only the client half is implemented: sing-box acts as a VLESS client.
package vlessenc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	cryptoRand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	E "github.com/sagernet/sing/common/exceptions"

	"golang.org/x/crypto/chacha20poly1305"
	"lukechampine.com/blake3"
)

// OutBytesPool holds record buffers: 5 byte fake-TLS header + 8192 byte max
// plaintext chunk + 16 byte AEAD tag.
var OutBytesPool = sync.Pool{
	New: func() any {
		return make([]byte, 5+8192+16)
	},
}

// CommonConn is the post-handshake record layer. Every record looks like a
// TLS 1.3 application data record on the wire: 17 03 03 <len> <ciphertext>.
type CommonConn struct {
	net.Conn
	UseAES      bool
	Client      *ClientInstance
	UnitedKey   []byte
	PreWrite    []byte
	AEAD        *AEAD
	PeerAEAD    *AEAD
	PeerPadding []byte
	rawInput    bytes.Buffer
	input       bytes.Reader
}

func NewCommonConn(conn net.Conn, useAES bool) *CommonConn {
	return &CommonConn{
		Conn:   conn,
		UseAES: useAES,
	}
}

func (c *CommonConn) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	outBytes := OutBytesPool.Get().([]byte)
	defer OutBytesPool.Put(outBytes)
	for n := 0; n < len(b); {
		b := b[n:]
		if len(b) > 8192 {
			b = b[:8192] // for avoiding another copy() in peer's Read()
		}
		n += len(b)
		headerAndData := outBytes[:5+len(b)+16]
		EncodeHeader(headerAndData, len(b)+16)
		max := false
		if bytes.Equal(c.AEAD.Nonce[:], MaxNonce) {
			max = true
		}
		c.AEAD.Seal(headerAndData[:5], nil, b, headerAndData[:5])
		if max {
			// Re-key with the last record as context once the nonce is exhausted.
			c.AEAD = NewAEAD(headerAndData, c.UnitedKey, c.UseAES)
		}
		if c.PreWrite != nil {
			// 0-RTT: the client hello is flushed together with the first record.
			headerAndData = append(c.PreWrite, headerAndData...)
			c.PreWrite = nil
		}
		if _, err := c.Conn.Write(headerAndData); err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (c *CommonConn) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	if c.PeerAEAD == nil { // client's 0-RTT
		serverRandom := make([]byte, 16)
		if _, err := io.ReadFull(c.Conn, serverRandom); err != nil {
			return 0, err
		}
		c.PeerAEAD = NewAEAD(serverRandom, c.UnitedKey, c.UseAES)
		if xorConn, ok := c.Conn.(*XorConn); ok {
			xorConn.PeerCTR = NewCTR(c.UnitedKey, serverRandom)
		}
	}
	if c.PeerPadding != nil { // client's 1-RTT
		if _, err := io.ReadFull(c.Conn, c.PeerPadding); err != nil {
			return 0, err
		}
		if _, err := c.PeerAEAD.Open(c.PeerPadding[:0], nil, c.PeerPadding, nil); err != nil {
			return 0, err
		}
		c.PeerPadding = nil
	}
	if c.input.Len() > 0 {
		return c.input.Read(b)
	}
	peerHeader := [5]byte{}
	if _, err := io.ReadFull(c.Conn, peerHeader[:]); err != nil {
		return 0, err
	}
	l, err := DecodeHeader(peerHeader[:]) // l: 17~16640
	if err != nil {
		if c.Client != nil && strings.Contains(err.Error(), "invalid header: ") { // client's 0-RTT
			c.Client.RWLock.Lock()
			if bytes.HasPrefix(c.UnitedKey, c.Client.PfsKey) {
				c.Client.Expire = time.Now() // expired
			}
			c.Client.RWLock.Unlock()
			return 0, ErrNewHandshakeNeeded
		}
		return 0, err
	}
	c.Client = nil
	if c.rawInput.Cap() < l {
		c.rawInput.Grow(l) // no need to use sync.Pool, because we are always reading
	}
	peerData := c.rawInput.Bytes()[:l]
	if _, err := io.ReadFull(c.Conn, peerData); err != nil {
		return 0, err
	}
	dst := peerData[:l-16]
	if len(dst) <= len(b) {
		dst = b[:len(dst)] // avoids another copy()
	}
	var newAEAD *AEAD
	if bytes.Equal(c.PeerAEAD.Nonce[:], MaxNonce) {
		newAEAD = NewAEAD(append(peerHeader[:], peerData...), c.UnitedKey, c.UseAES)
	}
	_, err = c.PeerAEAD.Open(dst[:0], nil, peerData, peerHeader[:])
	if newAEAD != nil {
		c.PeerAEAD = newAEAD
	}
	if err != nil {
		return 0, err
	}
	if len(dst) > len(b) {
		c.input.Reset(dst[copy(b, dst):])
		dst = b // for len(dst)
	}
	return len(dst), nil
}

// Upstream returns the connection below the record layer (the raw or XOR
// wrapped connection). Used by sing-box to detect splice eligibility.
func (c *CommonConn) Upstream() any {
	return c.Conn
}

// NeedHandshake reports nothing useful here but keeps the conn from being
// treated as a plain TCP conn by sing's splice detection.
func (c *CommonConn) NeedAdditionalReadDeadline() bool {
	return true
}

// ErrNewHandshakeNeeded is returned when the server rejected our 0-RTT ticket.
// The caller should redial and perform a full 1-RTT handshake.
var ErrNewHandshakeNeeded = E.New("new handshake needed")

type AEAD struct {
	cipher.AEAD
	Nonce [12]byte
}

// NewAEAD derives a fresh AEAD from key using ctx as BLAKE3 derive_key context.
func NewAEAD(ctx, key []byte, useAES bool) *AEAD {
	k := make([]byte, 32)
	blake3.DeriveKey(k, string(ctx), key)
	var aead cipher.AEAD
	if useAES {
		block, _ := aes.NewCipher(k)
		aead, _ = cipher.NewGCM(block)
	} else {
		aead, _ = chacha20poly1305.New(k)
	}
	return &AEAD{AEAD: aead}
}

func (a *AEAD) Seal(dst, nonce, plaintext, additionalData []byte) []byte {
	if nonce == nil {
		nonce = IncreaseNonce(a.Nonce[:])
	}
	return a.AEAD.Seal(dst, nonce, plaintext, additionalData)
}

func (a *AEAD) Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	if nonce == nil {
		nonce = IncreaseNonce(a.Nonce[:])
	}
	return a.AEAD.Open(dst, nonce, ciphertext, additionalData)
}

// IncreaseNonce increments the 12 byte counter from the last byte down, so the
// very first nonce used by a fresh AEAD is 00..01.
func IncreaseNonce(nonce []byte) []byte {
	for i := range 12 {
		nonce[11-i]++
		if nonce[11-i] != 0 {
			break
		}
	}
	return nonce
}

// MaxNonce is both the re-key trigger and the fixed nonce used for the
// server's PFS public key block.
var MaxNonce = bytes.Repeat([]byte{255}, 12)

func EncodeLength(l int) []byte {
	return []byte{byte(l >> 8), byte(l)}
}

func DecodeLength(b []byte) int {
	return int(b[0])<<8 | int(b[1])
}

func EncodeHeader(h []byte, l int) {
	h[0] = 23
	h[1] = 3
	h[2] = 3
	h[3] = byte(l >> 8)
	h[4] = byte(l)
}

func DecodeHeader(h []byte) (l int, err error) {
	l = int(h[3])<<8 | int(h[4])
	if h[0] != 23 || h[1] != 3 || h[2] != 3 {
		l = 0
	}
	if l < 17 || l > 16640 { // TLS 1.3 max record: 16384 + 256 (RFC 8446 §5.2)
		err = E.New("invalid header: " + fmt.Sprintf("%v", h[:5])) // DO NOT CHANGE: relied by client's Read()
	}
	return
}

// ParsePadding parses Xray's padding DSL: dot separated "chance-min-max"
// triplets, where even elements are lengths and odd ones are time gaps in ms.
func ParsePadding(padding string, paddingLens, paddingGaps *[][3]int) (err error) {
	if padding == "" {
		return
	}
	maxLen := 0
	for i, s := range strings.Split(padding, ".") {
		x := strings.Split(s, "-")
		if len(x) < 3 || x[0] == "" || x[1] == "" || x[2] == "" {
			return E.New("invalid padding lenth/gap parameter: " + s)
		}
		y := [3]int{}
		if y[0], err = strconv.Atoi(x[0]); err != nil {
			return
		}
		if y[1], err = strconv.Atoi(x[1]); err != nil {
			return
		}
		if y[2], err = strconv.Atoi(x[2]); err != nil {
			return
		}
		if i == 0 && (y[0] < 100 || y[1] < 18+17 || y[2] < 18+17) {
			return E.New("first padding length must not be smaller than 35")
		}
		if i%2 == 0 {
			*paddingLens = append(*paddingLens, y)
			maxLen += max(y[1], y[2])
		} else {
			*paddingGaps = append(*paddingGaps, y)
		}
	}
	if maxLen > 18+65535 {
		return E.New("total padding length must not be larger than 65553")
	}
	return
}

// CreatPadding rolls the actual padding lengths and gaps for one handshake.
// Name kept as in Xray for easier diffing.
func CreatPadding(paddingLens, paddingGaps [][3]int) (length int, lens []int, gaps []time.Duration) {
	if len(paddingLens) == 0 {
		paddingLens = [][3]int{{100, 111, 1111}, {50, 0, 3333}}
		paddingGaps = [][3]int{{75, 0, 111}}
	}
	for _, y := range paddingLens {
		l := 0
		if y[0] >= int(RandBetween(0, 100)) {
			l = int(RandBetween(int64(y[1]), int64(y[2])))
		}
		lens = append(lens, l)
		length += l
	}
	for _, y := range paddingGaps {
		g := 0
		if y[0] >= int(RandBetween(0, 100)) {
			g = int(RandBetween(int64(y[1]), int64(y[2])))
		}
		gaps = append(gaps, time.Duration(g)*time.Millisecond)
	}
	return
}

// RandBetween returns a random value in [from, to), matching Xray's
// common/crypto.RandBetween semantics (including the degenerate cases).
func RandBetween(from int64, to int64) int64 {
	if from == to {
		return from
	}
	if from > to {
		from, to = to, from
	}
	d := to - from
	if d == 0 || d == 1 {
		return from
	}
	var buffer [8]byte
	if _, err := cryptoRand.Read(buffer[:]); err != nil {
		return from
	}
	return from + int64(binary.BigEndian.Uint64(buffer[:])%uint64(d))
}

// This file adapts configuration behavior from Xray-core 26.9.9
// (proxy/vless/encryption), licensed under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core
// See LICENSES/MPL-2.0.txt and THIRD_PARTY_NOTICES.md.

package vlessenc

import (
	"encoding/base64"
	"strings"

	E "github.com/sagernet/sing/common/exceptions"
)

// NewClient parses Xray's VLESS outbound "encryption" string and returns a
// ready to use ClientInstance. Returns (nil, nil) when encryption is disabled
// ("" or "none").
//
// Grammar (identical to Xray-core 26.9.9):
//
//	mlkem768x25519plus.<native|xorpub|random>.<1rtt|0rtt>[.<padding>].<key>...
//
// where <padding> is a dot separated list of "chance-min-max" elements (each
// shorter than 20 characters) and every <key> is base64 raw-url encoded and is
// either 32 bytes (X25519 public key) or 1184 bytes (ML-KEM-768 encapsulation
// key). Multiple keys describe a relay chain, in order.
//
// Example:
//
//	mlkem768x25519plus.native.0rtt.100-111-1111.75-0-111.50-0-3333.<base64 key>
func NewClient(encryption string) (*ClientInstance, error) {
	switch encryption {
	case "", "none":
		return nil, nil
	}

	s := strings.Split(encryption, ".")
	if len(s) < 4 || s[0] != "mlkem768x25519plus" {
		return nil, E.New("unsupported VLESS encryption: ", encryption)
	}

	var xorMode uint32
	switch s[1] {
	case "native":
		xorMode = 0
	case "xorpub":
		xorMode = 1
	case "random":
		xorMode = 2
	default:
		return nil, E.New("unsupported VLESS encryption XOR mode: ", s[1])
	}

	var seconds uint32
	switch s[2] {
	case "1rtt":
		seconds = 0
	case "0rtt":
		seconds = 1
	default:
		return nil, E.New("unsupported VLESS encryption RTT mode: ", s[2])
	}

	// Elements shorter than 20 characters are padding parameters, the rest are
	// keys. Xray measures the padding prefix in characters, then re-slices the
	// original string, so we do exactly the same.
	padding := 0
	for _, r := range s[3:] {
		if len(r) < 20 {
			padding += len(r) + 1
			continue
		}
		if b, _ := base64.RawURLEncoding.DecodeString(r); len(b) != 32 && len(b) != 1184 {
			return nil, E.New("invalid VLESS encryption key length: ", r)
		}
	}
	rest := encryption[27+len(s[2]):]
	var paddingString string
	if padding > 0 {
		paddingString = rest[:padding-1]
		rest = rest[padding:]
	}

	var nfsPKeysBytes [][]byte
	for _, r := range strings.Split(rest, ".") {
		if r == "" {
			continue
		}
		b, err := base64.RawURLEncoding.DecodeString(r)
		if err != nil {
			return nil, E.Cause(err, "invalid VLESS encryption key: ", r)
		}
		if len(b) != 32 && len(b) != 1184 {
			return nil, E.New("invalid VLESS encryption key length: ", len(b))
		}
		nfsPKeysBytes = append(nfsPKeysBytes, b)
	}
	if len(nfsPKeysBytes) == 0 {
		return nil, E.New("empty VLESS encryption keys")
	}

	client := new(ClientInstance)
	if err := client.Init(nfsPKeysBytes, xorMode, seconds, paddingString); err != nil {
		return nil, E.Cause(err, "failed to use encryption")
	}
	return client, nil
}

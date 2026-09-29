// Helpers ported from Xray-core (common/crypto, common/uuid), licensed under
// the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core (v26.9.9)
//
// Modifications for sing-box are licensed under GPL-3.0-or-later.

package v2rayxhttp

import (
	"bytes"
	crand "crypto/rand"
	"io"
	"math/big"
	"net/http"

	"github.com/gofrs/uuid/v5"
)

// randBetween mirrors Xray's crypto.RandBetween.
//
// IMPORTANT: the interval is half-open [from, to). A range of {From: 1, To: 2}
// therefore always yields 1. Ranges in XHTTP configs inherit this behaviour,
// so it must not be "fixed" to be inclusive.
func randBetween(from int64, to int64) int64 {
	if from > to {
		from, to = to, from
	}
	if d := to - from; d == 0 || d == 1 {
		return from
	}
	bigInt, _ := crand.Int(crand.Reader, big.NewInt(to-from))
	return from + bigInt.Int64()
}

// newUUIDString is the fallback session ID: a v4 UUID in canonical 36-char
// dashed form, matching Xray's uuid.New().String().
func newUUIDString() string {
	id, err := uuid.NewV4()
	if err != nil {
		// uuid.NewV4 only fails if the system CSPRNG fails.
		var buf [16]byte
		crand.Read(buf[:])
		id = uuid.UUID(buf)
	}
	return id.String()
}

// setRequestBody attaches a replayable in-memory body to a request.
func setRequestBody(request *http.Request, data []byte) {
	request.Body = io.NopCloser(bytes.NewReader(data))
	request.ContentLength = int64(len(data))
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
}

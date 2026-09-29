// This file adapts behavior from Xray-core 26.9.9
// (common/protocol), licensed under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core
// See LICENSES/MPL-2.0.txt and THIRD_PARTY_NOTICES.md.

package vlessenc

import (
	"runtime"

	"golang.org/x/sys/cpu"
)

// HasAESGCMHardwareSupport mirrors Xray's common/protocol.HasAESGCMHardwareSupport,
// which itself mirrors crypto/tls cipher suite preference logic. It decides
// whether the record layer uses AES-256-GCM (true) or ChaCha20-Poly1305 (false).
//
// This is a local decision on both sides: client and server each pick based on
// their own CPU, so the peers may legitimately differ.
var HasAESGCMHardwareSupport = func() bool {
	switch runtime.GOARCH {
	case "amd64":
		return cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ
	case "arm64":
		return cpu.ARM64.HasAES && cpu.ARM64.HasPMULL
	case "s390x":
		return cpu.S390X.HasAES && cpu.S390X.HasAESCBC && cpu.S390X.HasAESCTR &&
			(cpu.S390X.HasGHASH || cpu.S390X.HasAESGCM)
	case "ppc64", "ppc64le":
		return true
	default:
		return false
	}
}()

// This file is derived from Xray-core (transport/internet/splithttp/common.go),
// licensed under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core (v26.9.9)
//
// Modifications for sing-box are licensed under GPL-3.0-or-later.

package v2rayxhttp

// Placement values. These strings are part of the configuration contract with
// Xray and must not be renamed.
const (
	PlacementQueryInHeader = "queryInHeader"
	PlacementCookie        = "cookie"
	PlacementHeader        = "header"
	PlacementQuery         = "query"
	PlacementPath          = "path"
	PlacementBody          = "body"
	PlacementAuto          = "auto"
)

// Transport modes.
const (
	ModeAuto      = "auto"
	ModePacketUp  = "packet-up"
	ModeStreamUp  = "stream-up"
	ModeStreamOne = "stream-one"
)

// Default keys. These are the exact names Xray puts on the wire.
const (
	DefaultXPaddingKey    = "x_padding"
	DefaultXPaddingHeader = "X-Padding"

	DefaultSessionKeyHeader = "X-Session"
	DefaultSessionKeyCookie = "x_session"

	DefaultSeqKeyHeader = "X-Seq"
	DefaultSeqKeyCookie = "x_seq"

	DefaultUplinkDataKeyHeader = "X-Data"
	DefaultUplinkDataKeyCookie = "x_data"
)

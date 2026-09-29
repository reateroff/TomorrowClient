package option

import (
	"strconv"
	"strings"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
)

// XHTTPRange is Xray's Int32Range. It accepts either a bare number (5) or a
// "from-to" string ("5-10").
//
// NOTE: the interval is half-open [from, to) at use time, exactly like Xray.
type XHTTPRange struct {
	From int32
	To   int32
}

func (r XHTTPRange) MarshalJSON() ([]byte, error) {
	if r.From == r.To {
		return json.Marshal(r.From)
	}
	return json.Marshal(strconv.Itoa(int(r.From)) + "-" + strconv.Itoa(int(r.To)))
}

func (r *XHTTPRange) UnmarshalJSON(data []byte) error {
	var number int32
	if err := json.Unmarshal(data, &number); err == nil {
		r.From = number
		r.To = number
		return nil
	}
	var rangeString string
	if err := json.Unmarshal(data, &rangeString); err != nil {
		return E.New("invalid range value: ", string(data))
	}
	rangeString = strings.TrimSpace(rangeString)
	pair := strings.SplitN(rangeString, "-", 2)
	from, err := strconv.Atoi(strings.TrimSpace(pair[0]))
	if err != nil {
		return E.Cause(err, "invalid range value: ", rangeString)
	}
	if len(pair) == 1 {
		r.From = int32(from)
		r.To = int32(from)
		return nil
	}
	to, err := strconv.Atoi(strings.TrimSpace(pair[1]))
	if err != nil {
		return E.Cause(err, "invalid range value: ", rangeString)
	}
	if from > to {
		from, to = to, from
	}
	r.From = int32(from)
	r.To = int32(to)
	return nil
}

// XHTTPMuxOptions maps to Xray's XmuxConfig.
type XHTTPMuxOptions struct {
	MaxConcurrency   XHTTPRange `json:"maxConcurrency,omitempty"`
	MaxConnections   XHTTPRange `json:"maxConnections,omitempty"`
	CMaxReuseTimes   XHTTPRange `json:"cMaxReuseTimes,omitempty"`
	HMaxRequestTimes XHTTPRange `json:"hMaxRequestTimes,omitempty"`
	HMaxReusableSecs XHTTPRange `json:"hMaxReusableSecs,omitempty"`
	HKeepAlivePeriod int64      `json:"hKeepAlivePeriod,omitempty"`
}

// V2RayXHTTPOptions maps to Xray's SplitHTTPConfig (XHTTP). Field names follow
// Xray's JSON so that existing XHTTP client configs can be reused verbatim.
type V2RayXHTTPOptions struct {
	Host                 string                     `json:"host,omitempty"`
	Path                 string                     `json:"path,omitempty"`
	Mode                 string                     `json:"mode,omitempty" enum:"auto,packet-up,stream-up,stream-one"`
	Headers              map[string]string          `json:"headers,omitempty"`
	XPaddingBytes        XHTTPRange                 `json:"xPaddingBytes,omitempty"`
	XPaddingObfsMode     bool                       `json:"xPaddingObfsMode,omitempty"`
	XPaddingKey          string                     `json:"xPaddingKey,omitempty"`
	XPaddingHeader       string                     `json:"xPaddingHeader,omitempty"`
	XPaddingPlacement    string                     `json:"xPaddingPlacement,omitempty"`
	XPaddingMethod       string                     `json:"xPaddingMethod,omitempty"`
	UplinkHTTPMethod     string                     `json:"uplinkHTTPMethod,omitempty"`
	SessionIDPlacement   string                     `json:"sessionIDPlacement,omitempty"`
	SessionIDKey         string                     `json:"sessionIDKey,omitempty"`
	SessionIDTable       string                     `json:"sessionIDTable,omitempty"`
	SessionIDLength      XHTTPRange                 `json:"sessionIDLength,omitempty"`
	SeqPlacement         string                     `json:"seqPlacement,omitempty"`
	SeqKey               string                     `json:"seqKey,omitempty"`
	UplinkDataPlacement  string                     `json:"uplinkDataPlacement,omitempty"`
	UplinkDataKey        string                     `json:"uplinkDataKey,omitempty"`
	UplinkChunkSize      XHTTPRange                 `json:"uplinkChunkSize,omitempty"`
	NoGRPCHeader         bool                       `json:"noGRPCHeader,omitempty"`
	NoSSEHeader          bool                       `json:"noSSEHeader,omitempty"`
	ScMaxEachPostBytes   XHTTPRange                 `json:"scMaxEachPostBytes,omitempty"`
	ScMinPostsIntervalMs XHTTPRange                 `json:"scMinPostsIntervalMs,omitempty"`
	ScMaxBufferedPosts   int64                      `json:"scMaxBufferedPosts,omitempty"`
	ScStreamUpServerSecs XHTTPRange                 `json:"scStreamUpServerSecs,omitempty"`
	ServerMaxHeaderBytes int32                      `json:"serverMaxHeaderBytes,omitempty"`
	Xmux                 *XHTTPMuxOptions           `json:"xmux,omitempty"`
	DownloadSettings     *V2RayXHTTPDownloadOptions `json:"downloadSettings,omitempty"`
}

// V2RayXHTTPDownloadOptions describes a separate downlink transport, as in
// Xray's "downloadSettings".
type V2RayXHTTPDownloadOptions struct {
	ServerOptions
	TLS *OutboundTLSOptions `json:"tls,omitempty"`
	V2RayXHTTPOptions
}

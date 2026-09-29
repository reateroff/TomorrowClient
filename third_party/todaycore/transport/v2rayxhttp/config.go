// This file is derived from Xray-core (transport/internet/splithttp/config.go
// and infra/conf/transport_method.go), licensed under the Mozilla Public
// License 2.0.
// Source: https://github.com/XTLS/Xray-core (v26.9.9)
//
// Modifications for sing-box are licensed under GPL-3.0-or-later.

package v2rayxhttp

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/tumgovic/todaycore/option"
	E "github.com/sagernet/sing/common/exceptions"

	mrand "math/rand/v2"
)

// RangeConfig is Xray's inclusive [From, To] integer range.
type RangeConfig struct {
	From int32
	To   int32
}

func (c *RangeConfig) rand() int32 {
	if c == nil {
		return 0
	}
	return int32(randBetween(int64(c.From), int64(c.To)))
}

type XmuxConfig struct {
	MaxConcurrency   *RangeConfig
	MaxConnections   *RangeConfig
	CMaxReuseTimes   *RangeConfig
	HMaxRequestTimes *RangeConfig
	HMaxReusableSecs *RangeConfig
	HKeepAlivePeriod int64
}

// Config is the runtime XHTTP configuration. Field names and semantics follow
// Xray's protobuf Config message exactly.
type Config struct {
	Host                 string
	Path                 string
	Mode                 string
	Headers              map[string]string
	XPaddingBytes        *RangeConfig
	NoGRPCHeader         bool
	NoSSEHeader          bool
	ScMaxEachPostBytes   *RangeConfig
	ScMinPostsIntervalMs *RangeConfig
	ScMaxBufferedPosts   int64
	ScStreamUpServerSecs *RangeConfig
	Xmux                 *XmuxConfig
	XPaddingObfsMode     bool
	XPaddingKey          string
	XPaddingHeader       string
	XPaddingPlacement    string
	XPaddingMethod       string
	UplinkHTTPMethod     string
	SessionIDPlacement   string
	SessionIDKey         string
	SeqPlacement         string
	SeqKey               string
	UplinkDataPlacement  string
	UplinkDataKey        string
	UplinkChunkSize      *RangeConfig
	ServerMaxHeaderBytes int32
	SessionIDTable       string
	SessionIDLength      *RangeConfig
}

func newRangeConfig(input option.XHTTPRange) *RangeConfig {
	return &RangeConfig{From: input.From, To: input.To}
}

func roomSize(tableSize int, minLen, maxLen int32) *big.Int {
	base := big.NewInt(int64(tableSize))
	sum := new(big.Int)
	term := new(big.Int)
	for k := minLen; k <= maxLen; k++ {
		term.Exp(base, big.NewInt(int64(k)), nil)
		sum.Add(sum, term)
	}
	return sum
}

// NewConfig validates options and applies Xray's Build() defaults. Error
// strings are kept identical to Xray so that existing troubleshooting
// knowledge transfers.
func NewConfig(options option.V2RayXHTTPOptions) (*Config, error) {
	c := options

	switch c.Mode {
	case "":
		c.Mode = ModeAuto
	case ModeAuto, ModePacketUp, ModeStreamUp, ModeStreamOne:
	default:
		return nil, E.New("unsupported mode: " + c.Mode)
	}

	// Priority (client): host > serverName > address
	for k := range c.Headers {
		if strings.ToLower(k) == "host" {
			return nil, E.New(`"headers" can't contain "host"`)
		}
	}

	if c.XPaddingBytes != (option.XHTTPRange{}) && (c.XPaddingBytes.From <= 0 || c.XPaddingBytes.To <= 0) {
		return nil, E.New("xPaddingBytes cannot be disabled")
	}

	if c.XPaddingKey == "" {
		c.XPaddingKey = DefaultXPaddingKey
	}
	if c.XPaddingHeader == "" {
		c.XPaddingHeader = DefaultXPaddingHeader
	}

	switch c.XPaddingPlacement {
	case "":
		c.XPaddingPlacement = PlacementQueryInHeader
	case PlacementCookie, PlacementHeader, PlacementQuery, PlacementQueryInHeader:
	default:
		return nil, E.New("unsupported padding placement: " + c.XPaddingPlacement)
	}

	switch c.XPaddingMethod {
	case "":
		c.XPaddingMethod = PaddingMethodRepeatX
	case PaddingMethodRepeatX, PaddingMethodTokenish:
	default:
		return nil, E.New("unsupported padding method: " + c.XPaddingMethod)
	}

	switch c.UplinkDataPlacement {
	case "":
		c.UplinkDataPlacement = PlacementAuto
	case PlacementAuto, PlacementBody:
	case PlacementCookie, PlacementHeader:
		if c.Mode != ModePacketUp {
			return nil, E.New("UplinkDataPlacement can be " + c.UplinkDataPlacement + " only in packet-up mode")
		}
	default:
		return nil, E.New("unsupported uplink data placement: " + c.UplinkDataPlacement)
	}

	if c.UplinkHTTPMethod == "" {
		c.UplinkHTTPMethod = http.MethodPost
	}
	c.UplinkHTTPMethod = strings.ToUpper(c.UplinkHTTPMethod)
	if c.UplinkHTTPMethod == http.MethodGet && c.Mode != ModePacketUp {
		return nil, E.New("uplinkHTTPMethod can be GET only in packet-up mode")
	}

	switch c.SessionIDPlacement {
	case "":
		c.SessionIDPlacement = PlacementPath
	case PlacementPath, PlacementCookie, PlacementHeader, PlacementQuery:
	default:
		return nil, E.New("unsupported session placement: " + c.SessionIDPlacement)
	}

	switch c.SeqPlacement {
	case "":
		c.SeqPlacement = PlacementPath
	case PlacementPath, PlacementCookie, PlacementHeader, PlacementQuery:
	default:
		return nil, E.New("unsupported seq placement: " + c.SeqPlacement)
	}

	if c.SessionIDPlacement != PlacementPath && c.SessionIDKey == "" {
		switch c.SessionIDPlacement {
		case PlacementCookie, PlacementQuery:
			c.SessionIDKey = DefaultSessionKeyCookie
		case PlacementHeader:
			c.SessionIDKey = DefaultSessionKeyHeader
		}
	}

	if c.SessionIDTable != "" {
		if predefined, ok := PredefinedTable[c.SessionIDTable]; ok {
			c.SessionIDTable = predefined
		}
		room := roomSize(len(c.SessionIDTable), c.SessionIDLength.From, c.SessionIDLength.To)
		// 2.1B possiblities should be enough
		if room.Cmp(big.NewInt(2<<30)) < 0 {
			return nil, E.New("sessionIDTable or sessionIDLength is too small")
		}
		if c.SessionIDLength.From <= 0 {
			return nil, E.New("sessionIDLength.from must be greater than 0")
		}
		for i := 0; i < len(c.SessionIDTable); i++ {
			if c.SessionIDTable[i] >= 0x80 {
				return nil, E.New("sessionIDTable must contain only ASCII characters")
			}
		}
	}

	if c.SeqPlacement != PlacementPath && c.SeqKey == "" {
		switch c.SeqPlacement {
		case PlacementCookie, PlacementQuery:
			c.SeqKey = DefaultSeqKeyCookie
		case PlacementHeader:
			c.SeqKey = DefaultSeqKeyHeader
		}
	}

	if c.UplinkDataPlacement != PlacementBody && c.UplinkDataKey == "" {
		switch c.UplinkDataPlacement {
		case PlacementCookie:
			c.UplinkDataKey = DefaultUplinkDataKeyCookie
		case PlacementAuto, PlacementHeader:
			c.UplinkDataKey = DefaultUplinkDataKeyHeader
		}
	}

	if c.ServerMaxHeaderBytes < 0 {
		return nil, E.New("invalid negative value of maxHeaderBytes")
	}

	var xmux option.XHTTPMuxOptions
	if c.Xmux != nil {
		xmux = *c.Xmux
	}
	if xmux.MaxConnections.To > 0 && xmux.MaxConcurrency.To > 0 {
		return nil, E.New("maxConnections cannot be specified together with maxConcurrency")
	}
	if xmux == (option.XHTTPMuxOptions{}) {
		xmux.MaxConnections.From = 3
		xmux.MaxConnections.To = 3
		xmux.HMaxRequestTimes.From = 600
		xmux.HMaxRequestTimes.To = 900
		xmux.HMaxReusableSecs.From = 1800
		xmux.HMaxReusableSecs.To = 3000
	}

	return &Config{
		Host:                 c.Host,
		Path:                 c.Path,
		Mode:                 c.Mode,
		Headers:              c.Headers,
		XPaddingBytes:        newRangeConfig(c.XPaddingBytes),
		XPaddingObfsMode:     c.XPaddingObfsMode,
		XPaddingKey:          c.XPaddingKey,
		XPaddingHeader:       c.XPaddingHeader,
		XPaddingPlacement:    c.XPaddingPlacement,
		XPaddingMethod:       c.XPaddingMethod,
		UplinkHTTPMethod:     c.UplinkHTTPMethod,
		SessionIDPlacement:   c.SessionIDPlacement,
		SeqPlacement:         c.SeqPlacement,
		SessionIDKey:         c.SessionIDKey,
		SeqKey:               c.SeqKey,
		UplinkDataPlacement:  c.UplinkDataPlacement,
		UplinkDataKey:        c.UplinkDataKey,
		UplinkChunkSize:      newRangeConfig(c.UplinkChunkSize),
		NoGRPCHeader:         c.NoGRPCHeader,
		NoSSEHeader:          c.NoSSEHeader,
		ScMaxEachPostBytes:   newRangeConfig(c.ScMaxEachPostBytes),
		ScMinPostsIntervalMs: newRangeConfig(c.ScMinPostsIntervalMs),
		ScMaxBufferedPosts:   c.ScMaxBufferedPosts,
		ScStreamUpServerSecs: newRangeConfig(c.ScStreamUpServerSecs),
		ServerMaxHeaderBytes: c.ServerMaxHeaderBytes,
		SessionIDTable:       c.SessionIDTable,
		SessionIDLength:      newRangeConfig(c.SessionIDLength),
		Xmux: &XmuxConfig{
			MaxConcurrency:   newRangeConfig(xmux.MaxConcurrency),
			MaxConnections:   newRangeConfig(xmux.MaxConnections),
			CMaxReuseTimes:   newRangeConfig(xmux.CMaxReuseTimes),
			HMaxRequestTimes: newRangeConfig(xmux.HMaxRequestTimes),
			HMaxReusableSecs: newRangeConfig(xmux.HMaxReusableSecs),
			HKeepAlivePeriod: xmux.HKeepAlivePeriod,
		},
	}, nil
}

func (c *Config) GetNormalizedPath() string {
	pathAndQuery := strings.SplitN(c.Path, "?", 2)
	path := pathAndQuery[0]

	if path == "" || path[0] != '/' {
		path = "/" + path
	}

	if c.GetNormalizedSessionPlacement() == PlacementPath ||
		c.GetNormalizedSeqPlacement() == PlacementPath {
		if path[len(path)-1] != '/' {
			path = path + "/"
		}
	}

	return path
}

func (c *Config) GetNormalizedQuery() string {
	pathAndQuery := strings.SplitN(c.Path, "?", 2)
	query := ""
	if len(pathAndQuery) > 1 {
		query = pathAndQuery[1]
	}
	return query
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	for k, v := range c.Headers {
		header.Add(k, v)
	}
	TryDefaultHeadersWith(header, "fetch")
	return header
}

func (c *Config) GetRequestHeaderWithPayload(payload []byte) http.Header {
	header := c.GetRequestHeader()

	key := c.UplinkDataKey
	encodedData := base64.RawURLEncoding.EncodeToString(payload)

	for i := 0; len(encodedData) > 0; i++ {
		chunkSize := min(int(c.GetNormalizedUplinkChunkSize().rand()), len(encodedData))
		chunk := encodedData[:chunkSize]
		encodedData = encodedData[chunkSize:]
		headerKey := fmt.Sprintf("%s-%d", key, i)
		header.Set(headerKey, chunk)
	}

	return header
}

func (c *Config) GetRequestCookiesWithPayload(payload []byte) []*http.Cookie {
	cookies := []*http.Cookie{}

	key := c.UplinkDataKey
	encodedData := base64.RawURLEncoding.EncodeToString(payload)

	for i := 0; len(encodedData) > 0; i++ {
		chunkSize := min(int(c.GetNormalizedUplinkChunkSize().rand()), len(encodedData))
		chunk := encodedData[:chunkSize]
		encodedData = encodedData[chunkSize:]
		cookieName := fmt.Sprintf("%s_%d", key, i)
		cookies = append(cookies, &http.Cookie{Name: cookieName, Value: chunk})
	}

	return cookies
}

func (c *Config) GetNormalizedUplinkHTTPMethod() string {
	if c.UplinkHTTPMethod == "" {
		return http.MethodPost
	}
	return c.UplinkHTTPMethod
}

func (c *Config) GetNormalizedXPaddingBytes() *RangeConfig {
	if c.XPaddingBytes == nil || c.XPaddingBytes.To == 0 {
		return &RangeConfig{From: 100, To: 1000}
	}
	return c.XPaddingBytes
}

func (c *Config) GetNormalizedScMaxEachPostBytes() *RangeConfig {
	if c.ScMaxEachPostBytes == nil || c.ScMaxEachPostBytes.To == 0 {
		return &RangeConfig{From: 1000000, To: 1000000}
	}
	return c.ScMaxEachPostBytes
}

func (c *Config) GetNormalizedScMinPostsIntervalMs() *RangeConfig {
	if c.ScMinPostsIntervalMs == nil || c.ScMinPostsIntervalMs.To == 0 {
		return &RangeConfig{From: 30, To: 30}
	}
	return c.ScMinPostsIntervalMs
}

func (c *Config) GetNormalizedScMaxBufferedPosts() int {
	if c.ScMaxBufferedPosts == 0 {
		return 30
	}
	return int(c.ScMaxBufferedPosts)
}

func (c *Config) GetNormalizedScStreamUpServerSecs() *RangeConfig {
	if c.ScStreamUpServerSecs == nil || c.ScStreamUpServerSecs.To == 0 {
		return &RangeConfig{From: 20, To: 80}
	}
	return c.ScStreamUpServerSecs
}

func (c *Config) GetNormalizedUplinkChunkSize() *RangeConfig {
	if c.UplinkChunkSize == nil || c.UplinkChunkSize.To == 0 {
		switch c.UplinkDataPlacement {
		case PlacementCookie:
			return &RangeConfig{From: 2 * 1024, To: 3 * 1024} // 2 KiB .. 3 KiB
		case PlacementHeader:
			return &RangeConfig{From: 3 * 1000, To: 4 * 1000} // 3 KB .. 4 KB
		default:
			return c.GetNormalizedScMaxEachPostBytes()
		}
	} else if c.UplinkChunkSize.From < 64 {
		return &RangeConfig{From: 64, To: max(64, c.UplinkChunkSize.To)}
	}
	return c.UplinkChunkSize
}

func (c *Config) GetNormalizedServerMaxHeaderBytes() int {
	if c.ServerMaxHeaderBytes <= 0 {
		return 8192
	}
	return int(c.ServerMaxHeaderBytes)
}

func (c *Config) GetNormalizedSessionPlacement() string {
	if c.SessionIDPlacement == "" {
		return PlacementPath
	}
	return c.SessionIDPlacement
}

func (c *Config) GetNormalizedSeqPlacement() string {
	if c.SeqPlacement == "" {
		return PlacementPath
	}
	return c.SeqPlacement
}

// GetNormalizedUplinkDataPlacement returns "body" for the empty value.
//
// NOTE: this intentionally differs from the JSON-layer default of "auto".
// Xray has the same split: infra/conf defaults to "auto", the runtime getter
// falls back to "body". Keep it.
func (c *Config) GetNormalizedUplinkDataPlacement() string {
	if c.UplinkDataPlacement == "" {
		return PlacementBody
	}
	return c.UplinkDataPlacement
}

func (c *Config) GetNormalizedSessionKey() string {
	if c.SessionIDKey != "" {
		return c.SessionIDKey
	}
	switch c.GetNormalizedSessionPlacement() {
	case PlacementHeader:
		return DefaultSessionKeyHeader
	case PlacementCookie, PlacementQuery:
		return DefaultSessionKeyCookie
	default:
		return ""
	}
}

func (c *Config) GetNormalizedSeqKey() string {
	if c.SeqKey != "" {
		return c.SeqKey
	}
	switch c.GetNormalizedSeqPlacement() {
	case PlacementHeader:
		return DefaultSeqKeyHeader
	case PlacementCookie, PlacementQuery:
		return DefaultSeqKeyCookie
	default:
		return ""
	}
}

// PredefinedTable maps sessionIDTable aliases to charsets.
var PredefinedTable = map[string]string{
	"ALPHABET": "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	"Alphabet": "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
	"BASE36":   "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	"Base62":   "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
	"HEX":      "0123456789ABCDEF",
	"alphabet": "abcdefghijklmnopqrstuvwxyz",
	"base36":   "0123456789abcdefghijklmnopqrstuvwxyz",
	"hex":      "0123456789abcdef",
	"number":   "0123456789",
}

func (c *Config) GenerateSessionID() string {
	length := c.SessionIDLength.rand()
	table := c.SessionIDTable
	if predefined, ok := PredefinedTable[table]; ok {
		table = predefined
	}
	if table != "" && length > 0 {
		id := make([]byte, length)
		for i := range id {
			id[i] = table[mrand.N(len(table))]
		}
		return string(id)
	}
	return newUUIDString()
}

func appendToPath(path, value string) string {
	if strings.HasSuffix(path, "/") {
		return path + value
	}
	return path + "/" + value
}

func (c *Config) ApplyMetaToRequest(req *http.Request, sessionId string, seqStr string) {
	sessionPlacement := c.GetNormalizedSessionPlacement()
	seqPlacement := c.GetNormalizedSeqPlacement()
	sessionKey := c.GetNormalizedSessionKey()
	seqKey := c.GetNormalizedSeqKey()

	if sessionId != "" {
		switch sessionPlacement {
		case PlacementPath:
			req.URL.Path = appendToPath(req.URL.Path, sessionId)
		case PlacementQuery:
			q := req.URL.Query()
			q.Set(sessionKey, sessionId)
			req.URL.RawQuery = q.Encode()
		case PlacementHeader:
			req.Header.Set(sessionKey, sessionId)
		case PlacementCookie:
			req.AddCookie(&http.Cookie{Name: sessionKey, Value: sessionId})
		}
	}

	if seqStr != "" {
		switch seqPlacement {
		case PlacementPath:
			req.URL.Path = appendToPath(req.URL.Path, seqStr)
		case PlacementQuery:
			q := req.URL.Query()
			q.Set(seqKey, seqStr)
			req.URL.RawQuery = q.Encode()
		case PlacementHeader:
			req.Header.Set(seqKey, seqStr)
		case PlacementCookie:
			req.AddCookie(&http.Cookie{Name: seqKey, Value: seqStr})
		}
	}
}

func (c *Config) paddingConfig(rawURL string) XPaddingConfig {
	length := int(c.GetNormalizedXPaddingBytes().rand())
	config := XPaddingConfig{Length: length}
	if c.XPaddingObfsMode {
		config.Placement = XPaddingPlacement{
			Placement: c.XPaddingPlacement,
			Key:       c.XPaddingKey,
			Header:    c.XPaddingHeader,
			RawURL:    rawURL,
		}
		config.Method = c.XPaddingMethod
	} else {
		config.Placement = XPaddingPlacement{
			Placement: PlacementQueryInHeader,
			Key:       DefaultXPaddingKey,
			Header:    "Referer",
			RawURL:    rawURL,
		}
	}
	return config
}

// FillStreamRequest prepares a stream-one / stream-up / stream-down request.
func (c *Config) FillStreamRequest(request *http.Request, sessionId string, seqStr string) {
	request.Header = c.GetRequestHeader()
	c.ApplyXPaddingToRequest(request, c.paddingConfig(request.URL.String()))
	c.ApplyMetaToRequest(request, sessionId, "")

	if request.Body != nil && !c.NoGRPCHeader { // stream-up/one
		request.Header.Set("Content-Type", "application/grpc")
	}
}

// FillPacketRequest prepares a packet-up request carrying payload.
func (c *Config) FillPacketRequest(request *http.Request, sessionId string, seqStr string, data []byte) {
	dataPlacement := c.GetNormalizedUplinkDataPlacement()

	if dataPlacement == PlacementBody || dataPlacement == PlacementAuto {
		request.Header = c.GetRequestHeader()
		setRequestBody(request, data)
	} else {
		switch dataPlacement {
		case PlacementHeader:
			request.Header = c.GetRequestHeaderWithPayload(data)
		case PlacementCookie:
			request.Header = c.GetRequestHeader()
			for _, cookie := range c.GetRequestCookiesWithPayload(data) {
				request.AddCookie(cookie)
			}
		}
	}

	c.ApplyXPaddingToRequest(request, c.paddingConfig(request.URL.String()))
	c.ApplyMetaToRequest(request, sessionId, seqStr)
}

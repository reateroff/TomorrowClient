// This file is derived from Xray-core (transport/internet/splithttp/dialer.go),
// licensed under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core (v26.9.9)
//
// Modifications for sing-box are licensed under GPL-3.0-or-later.
//
// Client-side XHTTP only. Inbound/server support is intentionally absent.

package v2rayxhttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tumgovic/todaycore/adapter"
	"github.com/tumgovic/todaycore/common/tls"
	"github.com/tumgovic/todaycore/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"golang.org/x/net/http2"
)

const (
	// Xray uses these in transport/internet. Kept identical so that the
	// connection lifetime profile on the wire is unchanged.
	connIdleTimeout         = 300 * time.Second
	chromeH2KeepAlivePeriod = 45 * time.Second
)

var (
	_ adapter.V2RayClientTransport          = (*Client)(nil)
	_ adapter.V2RayMultiplexClientTransport = (*Client)(nil)
)

type Client struct {
	ctx         context.Context
	dialer      N.Dialer
	serverAddr  M.Socksaddr
	tlsConfig   tls.Config
	logger      logger.ContextLogger
	config      *Config
	httpVersion string
	reality     bool

	access      sync.Mutex
	xmuxManager *XmuxManager

	// Separate manager for the downlink of stream-up mode, mirroring Xray's
	// DownloadSettings having its own dialer state.
	download       *Client
	isDownloadOnly bool
}

func NewClient(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options option.V2RayXHTTPOptions, tlsConfig tls.Config) (adapter.V2RayClientTransport, error) {
	config, err := NewConfig(options)
	if err != nil {
		return nil, err
	}
	clientLogger := logger.NOP()
	client, err := newClient(ctx, clientLogger, dialer, serverAddr, config, tlsConfig, false)
	if err != nil {
		return nil, err
	}
	if options.DownloadSettings != nil {
		if config.Mode == ModeStreamOne {
			return nil, E.New(`Can not use "downloadSettings" in "stream-one" mode.`)
		}
		downloadClient, err := newDownloadClient(ctx, clientLogger, dialer, *options.DownloadSettings)
		if err != nil {
			return nil, E.Cause(err, "downloadSettings")
		}
		client.download = downloadClient
	}
	return client, nil
}

func newClient(ctx context.Context, clientLogger logger.ContextLogger, dialer N.Dialer, serverAddr M.Socksaddr, config *Config, tlsConfig tls.Config, downloadOnly bool) (*Client, error) {
	reality := isRealityConfig(tlsConfig)
	httpVersion, err := decideHTTPVersion(tlsConfig, reality)
	if err != nil {
		return nil, err
	}
	if httpVersion == "2" && tlsConfig != nil && len(tlsConfig.NextProtos()) == 0 {
		tlsConfig.SetNextProtos([]string{http2.NextProtoTLS})
	}
	client := &Client{
		ctx:            ctx,
		logger:         clientLogger,
		dialer:         dialer,
		serverAddr:     serverAddr,
		tlsConfig:      tlsConfig,
		config:         config,
		httpVersion:    httpVersion,
		reality:        reality,
		isDownloadOnly: downloadOnly,
	}
	client.xmuxManager = NewXmuxManager(*config.Xmux, clientLogger, func() XmuxConn {
		return client.createHTTPClient()
	})
	return client, nil
}

// newDownloadClient builds the separate downlink transport described by
// "downloadSettings". Xray allows a completely different server, TLS setting
// and XHTTP config for the downlink.
func newDownloadClient(ctx context.Context, clientLogger logger.ContextLogger, dialer N.Dialer, options option.V2RayXHTTPDownloadOptions) (*Client, error) {
	serverAddr := options.ServerOptions.Build()
	if !serverAddr.IsValid() {
		return nil, E.New("missing download server address")
	}
	var tlsConfig tls.Config
	if options.TLS != nil && options.TLS.Enabled {
		var err error
		tlsConfig, err = tls.NewClient(ctx, clientLogger, serverAddr.AddrString(), *options.TLS)
		if err != nil {
			return nil, err
		}
	}
	config, err := NewConfig(options.V2RayXHTTPOptions)
	_ = config
	if err != nil {
		return nil, err
	}
	return newClient(ctx, clientLogger, dialer, serverAddr, config, tlsConfig, true)
}

// isRealityConfig reports whether the TLS client is REALITY.
//
// sing-box exposes no interface method for this, and REALITY lives behind the
// with_reality build tag, so a direct type reference would break tag-less
// builds. The type name check is the only tag-independent option.
func isRealityConfig(tlsConfig tls.Config) bool {
	if tlsConfig == nil {
		return false
	}
	return strings.Contains(strings.ToLower(reflect.TypeOf(tlsConfig).String()), "reality")
}

// decideHTTPVersion follows Xray exactly, except that HTTP/3 is rejected:
// Xray's H3 path depends on its quic-go fork, which sing-box does not carry.
func decideHTTPVersion(tlsConfig tls.Config, reality bool) (string, error) {
	if reality {
		return "2", nil
	}
	if tlsConfig == nil {
		return "1.1", nil
	}
	nextProtos := tlsConfig.NextProtos()
	if len(nextProtos) != 1 {
		return "2", nil
	}
	switch nextProtos[0] {
	case "http/1.1":
		return "1.1", nil
	case "h3":
		return "", E.New("XHTTP over HTTP/3 is not supported by sing-box: set tls.alpn to h2 or http/1.1")
	default:
		return "2", nil
	}
}

func (c *Client) MultiplexEnabled() bool {
	// Only HTTP/2 multiplexes; HTTP/1.1 uses one connection per stream.
	return c.httpVersion == "2"
}

func (c *Client) scheme() string {
	if c.tlsConfig != nil {
		return "https"
	}
	return "http"
}

func (c *Client) hostHeader() string {
	if c.config.Host != "" {
		return c.config.Host
	}
	if c.tlsConfig != nil && c.tlsConfig.ServerName() != "" {
		return c.tlsConfig.ServerName()
	}
	return c.serverAddr.String()
}

func (c *Client) requestURL() url.URL {
	return url.URL{
		Scheme:   c.scheme(),
		Host:     c.hostHeader(),
		Path:     c.config.GetNormalizedPath(),
		RawQuery: c.config.GetNormalizedQuery(),
	}
}

func (c *Client) dialContext(ctx context.Context) (net.Conn, error) {
	return c.dialer.DialContext(ctx, N.NetworkTCP, c.serverAddr)
}

func (c *Client) dialTLSContext(ctx context.Context) (net.Conn, error) {
	if c.tlsConfig == nil {
		return c.dialContext(ctx)
	}
	return tls.NewDialer(c.dialer, c.tlsConfig).DialTLSContext(ctx, c.serverAddr)
}

func (c *Client) createHTTPClient() *DefaultDialerClient {
	var keepAlivePeriod time.Duration
	switch period := c.config.Xmux.HKeepAlivePeriod; {
	case period > 0:
		keepAlivePeriod = time.Duration(period) * time.Second
	case period < 0:
		keepAlivePeriod = 0
	default:
		keepAlivePeriod = chromeH2KeepAlivePeriod
	}

	dialerClient := &DefaultDialerClient{
		transportConfig: c.config,
		httpVersion:     c.httpVersion,
		logger:          c.logger,
	}

	switch c.httpVersion {
	case "2":
		dialerClient.client = &http.Client{
			Transport: &http2.Transport{
				DialTLSContext: func(ctxInner context.Context, network string, addr string, cfg *tls.STDConfig) (net.Conn, error) {
					return c.dialTLSContext(ctxInner)
				},
				IdleConnTimeout: connIdleTimeout,
				ReadIdleTimeout: keepAlivePeriod,
			},
		}
	default: // "1.1"
		dialerClient.client = &http.Client{
			Transport: &http.Transport{
				DialTLSContext: func(ctxInner context.Context, network string, addr string) (net.Conn, error) {
					return c.dialTLSContext(ctxInner)
				},
				DialContext: func(ctxInner context.Context, network string, addr string) (net.Conn, error) {
					return c.dialContext(ctxInner)
				},
				IdleConnTimeout: connIdleTimeout,
				// Chrome's H1 behaviour for this kind of streaming.
				DisableKeepAlives: true,
			},
		}
		dialerClient.uploadRawPool = &sync.Pool{}
		dialerClient.dialUploadConn = func(ctxInner context.Context) (net.Conn, error) {
			return c.dialTLSContext(ctxInner)
		}
	}
	return dialerClient
}

func (c *Client) getHTTPClient(ctx context.Context) (*DefaultDialerClient, *XmuxClient) {
	c.access.Lock()
	defer c.access.Unlock()
	xmuxClient := c.xmuxManager.GetXmuxClient(ctx)
	return xmuxClient.XmuxConn.(*DefaultDialerClient), xmuxClient
}

func (c *Client) DialContext(ctx context.Context) (net.Conn, error) {
	mode := c.config.Mode
	if mode == ModeAuto {
		mode = ModePacketUp
		if c.reality {
			mode = ModeStreamOne
			if c.download != nil {
				mode = ModeStreamUp
			}
		}
	}

	sessionId := ""
	if mode != ModeStreamOne {
		sessionId = c.config.GenerateSessionID()
	}
	if c.logger != nil {
		c.logger.DebugContext(ctx, "XHTTP is dialing to ", c.serverAddr, ", mode ", mode, ", sessionId ", sessionId)
	}

	requestURL := c.requestURL()
	urlString := requestURL.String()

	httpClient, xmuxClient := c.getHTTPClient(ctx)
	xmuxClient.AddRunning()
	xmuxClient.LeftRequests.Add(-1)

	if mode == ModeStreamOne {
		pipe := newUploadPipe(0)
		reader, remoteAddr, localAddr, err := httpClient.OpenStream(ctx, urlString, sessionId, &pipeReader{pipe: pipe}, false)
		if err != nil {
			xmuxClient.DoneRunning()
			return nil, err
		}
		return &splitConn{
			writer:     pipe,
			reader:     reader,
			remoteAddr: remoteAddr,
			localAddr:  localAddr,
			onClose:    xmuxClient.DoneRunning,
		}, nil
	}

	// Downlink first: the server needs the GET before it can answer uploads.
	downClient, downXmux := httpClient, xmuxClient
	downURL := urlString
	if c.download != nil {
		downClient, downXmux = c.download.getHTTPClient(ctx)
		downXmux.AddRunning()
		downXmux.LeftRequests.Add(-1)
		downRequestURL := c.download.requestURL()
		downURL = downRequestURL.String()
	}

	reader, remoteAddr, localAddr, err := downClient.OpenStream(ctx, downURL, sessionId, nil, false)
	if err != nil {
		xmuxClient.DoneRunning()
		if downXmux != xmuxClient {
			downXmux.DoneRunning()
		}
		return nil, err
	}

	onClose := func() {
		xmuxClient.DoneRunning()
		if downXmux != xmuxClient {
			downXmux.DoneRunning()
		}
	}

	if mode == ModeStreamUp {
		pipe := newUploadPipe(0)
		_, _, _, err = httpClient.OpenStream(ctx, urlString, sessionId, &pipeReader{pipe: pipe}, true)
		if err != nil {
			reader.Close()
			onClose()
			return nil, err
		}
		return &splitConn{
			writer:     pipe,
			reader:     reader,
			remoteAddr: remoteAddr,
			localAddr:  localAddr,
			onClose:    onClose,
		}, nil
	}

	// packet-up
	maxUploadSize := int(c.config.GetNormalizedScMaxEachPostBytes().rand())
	// Reserve room so a single POST never exceeds maxUploadSize.
	pipe := newUploadPipe(maxUploadSize)
	uploadCtx, uploadCancel := context.WithCancel(context.WithoutCancel(ctx))

	go func() {
		defer uploadCancel()
		var seq int64
		var lastWrite time.Time
		activeClient, activeXmux := httpClient, xmuxClient

		for {
			chunk, err := pipe.ReadChunk(maxUploadSize)
			if err != nil {
				break
			}

			if interval := time.Duration(c.config.GetNormalizedScMinPostsIntervalMs().rand()) * time.Millisecond; interval > 0 {
				if sleep := interval - time.Since(lastWrite); sleep > 0 {
					select {
					case <-time.After(sleep):
					case <-uploadCtx.Done():
						return
					}
				}
			}

			// Rotate the underlying HTTP client when its budget is exhausted.
			if activeXmux.LeftRequests.Add(-1) <= 0 ||
				(activeXmux.UnreusableAt != time.Time{} && lastWrite.After(activeXmux.UnreusableAt)) {
				newClient, newXmux := c.getHTTPClient(ctx)
				newXmux.AddRunning()
				if activeXmux != xmuxClient {
					activeXmux.DoneRunning()
				}
				activeClient, activeXmux = newClient, newXmux
			}

			seqStr := strconv.FormatInt(seq, 10)
			seq++
			lastWrite = time.Now()

			if err := activeClient.PostPacket(uploadCtx, urlString, sessionId, seqStr, chunk); err != nil {
				if c.logger != nil {
					c.logger.DebugContext(ctx, E.Cause(err, "failed to send packet ", seqStr))
				}
				pipe.Interrupt(err)
				reader.Close()
				break
			}
		}
		if activeXmux != xmuxClient {
			activeXmux.DoneRunning()
		}
	}()

	return &splitConn{
		writer:     pipe,
		reader:     reader,
		remoteAddr: remoteAddr,
		localAddr:  localAddr,
		onClose: func() {
			uploadCancel()
			onClose()
		},
	}, nil
}

func (c *Client) Close() error {
	c.access.Lock()
	defer c.access.Unlock()
	if c.download != nil {
		c.download.Close()
	}
	return nil
}

// pipeReader adapts uploadPipe to io.Reader for streaming request bodies.
type pipeReader struct {
	pipe      *uploadPipe
	remaining []byte
}

func (r *pipeReader) Read(b []byte) (int, error) {
	if len(r.remaining) == 0 {
		chunk, err := r.pipe.ReadChunk(len(b))
		if err != nil {
			return 0, err
		}
		r.remaining = chunk
	}
	n := copy(b, r.remaining)
	r.remaining = r.remaining[n:]
	return n, nil
}

func (r *pipeReader) Close() error {
	return r.pipe.Close()
}

var _ io.ReadCloser = (*pipeReader)(nil)

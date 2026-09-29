// This file is derived from Xray-core (transport/internet/splithttp/client.go),
// licensed under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core (v26.9.9)
//
// Modifications for sing-box are licensed under GPL-3.0-or-later.

package v2rayxhttp

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
)

// DialerClient abstracts over the HTTP versions used to carry XHTTP.
type DialerClient interface {
	IsClosed() bool

	// ctx, url, sessionId, body, uploadOnly
	OpenStream(context.Context, string, string, io.Reader, bool) (io.ReadCloser, net.Addr, net.Addr, error)

	// ctx, url, sessionId, seqStr, payload
	PostPacket(context.Context, string, string, string, []byte) error
}

// DefaultDialerClient implements DialerClient over direct network connections.
type DefaultDialerClient struct {
	transportConfig *Config
	client          *http.Client
	closed          atomic.Bool
	httpVersion     string
	logger          logger.ContextLogger

	// pool of net.Conn, created using dialUploadConn
	uploadRawPool  *sync.Pool
	dialUploadConn func(ctxInner context.Context) (net.Conn, error)
}

func (c *DefaultDialerClient) IsClosed() bool {
	return c.closed.Load()
}

func (c *DefaultDialerClient) OpenStream(ctx context.Context, urlString string, sessionId string, body io.Reader, uploadOnly bool) (wrc io.ReadCloser, remoteAddr, localAddr net.Addr, err error) {
	// Unblocks Dial() as soon as the TCP/UDP connection to the server is
	// established, so that correct addresses can be logged.
	gotConn := make(chan struct{})
	var gotConnOnce sync.Once
	closeGotConn := func() { gotConnOnce.Do(func() { close(gotConn) }) }

	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			remoteAddr = connInfo.Conn.RemoteAddr()
			localAddr = connInfo.Conn.LocalAddr()
			closeGotConn()
		},
	})

	method := http.MethodGet // stream-down
	if body != nil {
		method = c.transportConfig.GetNormalizedUplinkHTTPMethod() // stream-up/one
	}
	req, err := http.NewRequestWithContext(context.WithoutCancel(ctx), method, urlString, body)
	if err != nil {
		return nil, nil, nil, E.Cause(err, "create HTTP request for ", urlString)
	}
	c.transportConfig.FillStreamRequest(req, sessionId, "")

	waitReader := NewWaitReadCloser()
	wrc = waitReader
	go func() {
		resp, err := c.client.Do(req)
		if err != nil {
			if !uploadOnly { // stream-down is enough
				c.closed.Store(true)
				if c.logger != nil {
					c.logger.DebugContext(ctx, E.Cause(err, "failed to ", method, " ", urlString))
				}
			}
			closeGotConn()
			closeIfCloser(body)
			waitReader.Close()
			return
		}
		if resp.StatusCode != http.StatusOK && !uploadOnly && c.logger != nil {
			c.logger.DebugContext(ctx, "unexpected status ", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK || uploadOnly { // stream-up
			io.Copy(io.Discard, resp.Body)
			// Closing immediately would also interrupt the upload.
			resp.Body.Close()
			closeIfCloser(body)
			closeGotConn()
			waitReader.Close()
			return
		}
		waitReader.Set(resp.Body)
	}()

	<-gotConn
	return
}

func (c *DefaultDialerClient) PostPacket(ctx context.Context, urlString string, sessionId string, seqStr string, payload []byte) error {
	method := c.transportConfig.GetNormalizedUplinkHTTPMethod()
	req, err := http.NewRequestWithContext(context.WithoutCancel(ctx), method, urlString, nil)
	if err != nil {
		return err
	}
	c.transportConfig.FillPacketRequest(req, sessionId, seqStr, payload)

	if c.httpVersion != "1.1" {
		resp, err := c.client.Do(req)
		if err != nil {
			c.closed.Store(true)
			return err
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return E.New("bad status code: ", resp.Status)
		}
		return nil
	}

	// Stringify the entire HTTP/1.1 request so it can be safely retried.
	// Calling req.Write twice would send an already-drained body.
	requestBuff := new(bytes.Buffer)
	requestBuff.Grow(512 + int(req.ContentLength))
	err = req.Write(requestBuff)
	if err != nil {
		return E.Cause(err, "serialize HTTP/1.1 request")
	}

	var uploadConn any
	var h1UploadConn *H1Conn

	for {
		uploadConn = c.uploadRawPool.Get()
		newConnection := uploadConn == nil
		if newConnection {
			newConn, err := c.dialUploadConn(context.WithoutCancel(ctx))
			if err != nil {
				return err
			}
			h1UploadConn = NewH1Conn(newConn)
			uploadConn = h1UploadConn
		} else {
			h1UploadConn = uploadConn.(*H1Conn)

			// Drain the response left over by the previous request on this
			// pooled connection before pipelining another one.
			if h1UploadConn.UnreadedResponsesCount > 0 {
				resp, err := http.ReadResponse(h1UploadConn.RespBufReader, req)
				if err != nil {
					c.closed.Store(true)
					h1UploadConn.Close()
					return E.Cause(err, "read response")
				}
				h1UploadConn.UnreadedResponsesCount--
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					h1UploadConn.Close()
					return E.New("got non-200 error response code: ", resp.StatusCode)
				}
			}
		}

		_, err := h1UploadConn.Write(requestBuff.Bytes())
		// If the write failed we try another connection from the pool, until
		// a write on a *new* connection fails. Failed writes to a pooled
		// connection are normal when it was closed in the meantime.
		if err == nil {
			h1UploadConn.UnreadedResponsesCount++
			break
		} else if newConnection {
			return err
		}
		h1UploadConn.Close()
	}

	c.uploadRawPool.Put(uploadConn)
	return nil
}

// Close releases transport-level resources. HTTP/1.1 and HTTP/2 transports
// close themselves.
func (c *DefaultDialerClient) Close() error {
	c.closed.Store(true)
	if transport, ok := c.client.Transport.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

func closeIfCloser(v any) {
	if closer, ok := v.(io.Closer); ok {
		closer.Close()
	}
}

// This file is derived from Xray-core (transport/internet/splithttp/h1_conn.go),
// licensed under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core (v26.9.9)
//
// Modifications for sing-box are licensed under GPL-3.0-or-later.

package v2rayxhttp

import (
	std_bufio "bufio"
	"net"
)

// H1Conn is a pooled HTTP/1.1 upload connection.
//
// UnreadedResponsesCount tracks responses that have been sent by the server
// but not yet consumed, which is what makes request pipelining possible.
//
// NOTE: in Xray 26.9.9 this counter is never incremented, so the pipelining
// branch in PostPacket is dead code there and every pooled connection is
// treated as having no pending response. We keep the field and DO increment
// it, because a server that answers every POST would otherwise desynchronise
// the connection on reuse.
type H1Conn struct {
	UnreadedResponsesCount int
	RespBufReader          *std_bufio.Reader
	net.Conn
}

func NewH1Conn(conn net.Conn) *H1Conn {
	return &H1Conn{
		RespBufReader: std_bufio.NewReader(conn),
		Conn:          conn,
	}
}

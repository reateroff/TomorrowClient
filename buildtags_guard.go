//go:build !with_gvisor || !with_quic || !with_utls || !with_clash_api

package main

// TomorrowClient embeds sing-box, and sing-box keeps whole feature families
// behind build tags. Without them the app still compiles, but silently loses
// the gvisor TUN stack, the QUIC protocols (Hysteria, Hysteria2, TUIC),
// REALITY/uTLS, and the log hook — breakage that would only surface at runtime
// against a real server.
//
// wails.json passes the tags through "build:tags", so `wails build` and
// `wails dev` are already correct. This guard exists for a bare `go build`:
// referencing an undefined symbol turns a silently crippled binary into a
// compile error that names the tags it needs.
func init() {
	build_TomorrowClient_with_tags_with_gvisor_with_quic_with_utls_with_clash_api()
}

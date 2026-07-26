//go:build windows

package vpn

import (
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"

	"TomorrowClient/internal/model"
)

// activeTunName is the WinTun adapter name in use for the current connection.
// It defaults to the built-in name and is overwritten by the engine on Connect,
// keeping the sing-box config and this stats matcher in agreement.
var activeTunName = model.DefaultTunName

// adapterBytes returns the received and sent octets of the adapter the core
// created, matched by its alias. Returns ok=false when it is not present.
//
// The interface table comes from winipcfg rather than a hand-rolled
// MIB_IF_ROW2. That struct is 1352 bytes of counters ending in an easily
// missed OutQLen, and getting its size wrong fails silently in the worst way:
// the first row still parses, every later row is read at a drifting offset, the
// adapter is never matched, and the UI shows a permanent 0 Б with no error
// anywhere. winipcfg's layout is pinned by offset assertions in its own tests.
func adapterBytes() (rx, tx uint64, ok bool) {
	rows, err := winipcfg.GetIfTable2Ex(winipcfg.MibIfEntryNormal)
	if err != nil {
		return 0, 0, false
	}
	for i := range rows {
		if rows[i].Alias() == activeTunName {
			return rows[i].InOctets, rows[i].OutOctets, true
		}
	}
	return 0, 0, false
}

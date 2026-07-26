//go:build windows

package vpn

import (
	"testing"

	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

// The interface table must be readable end to end, not just its first row. A
// wrong row stride used to leave every later alias garbled, which is how the
// traffic counters silently reported zero forever.
func TestAdapterTableIsReadable(t *testing.T) {
	rows, err := winipcfg.GetIfTable2Ex(winipcfg.MibIfEntryNormal)
	if err != nil {
		t.Fatalf("GetIfTable2Ex: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("no interfaces returned")
	}

	named := 0
	for i := range rows {
		alias := rows[i].Alias()
		if alias == "" {
			continue
		}
		named++
		for _, r := range alias {
			if r == 0xFFFD || r > 0x2000 {
				t.Errorf("row %d alias looks misparsed: %q", i, alias)
				break
			}
		}
	}
	t.Logf("%d interfaces, %d named", len(rows), named)
	if named < 2 {
		t.Errorf("only %d rows carried a readable alias", named)
	}
}

// adapterBytes must find an adapter that exists and report its counters.
func TestAdapterBytesFindsRealAdapter(t *testing.T) {
	rows, err := winipcfg.GetIfTable2Ex(winipcfg.MibIfEntryNormal)
	if err != nil {
		t.Fatalf("GetIfTable2Ex: %v", err)
	}

	// Pick whichever adapter has actually carried traffic.
	var target string
	for i := range rows {
		if rows[i].Alias() != "" && rows[i].InOctets > 0 {
			target = rows[i].Alias()
			break
		}
	}
	if target == "" {
		t.Skip("no adapter with traffic on this machine")
	}

	saved := activeTunName
	defer func() { activeTunName = saved }()
	activeTunName = target

	rx, tx, ok := adapterBytes()
	if !ok {
		t.Fatalf("adapterBytes did not find %q", target)
	}
	t.Logf("%s rx=%d tx=%d", target, rx, tx)
	if rx == 0 {
		t.Error("counters read as zero for an adapter that has traffic")
	}
}

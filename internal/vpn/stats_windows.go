//go:build windows

package vpn

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// This file reads the byte counters of the WinTun adapter directly from the
// Windows IP Helper API (GetIfTable2). Both cores create an adapter named
// TomorrowTun, so a single implementation covers xray and sing-box.

var (
	modIphlpapi         = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetIfTable2     = modIphlpapi.NewProc("GetIfTable2")
	procFreeMibTable    = modIphlpapi.NewProc("FreeMibTable")
)

// mibIfRow2 mirrors the fields of MIB_IF_ROW2 we need. The struct must match
// the native layout, so unused fields are kept as padding arrays.
type mibIfRow2 struct {
	InterfaceLuid        uint64
	InterfaceIndex       uint32
	InterfaceGuid        [16]byte
	Alias                [257]uint16
	Description          [257]uint16
	PhysicalAddressLength uint32
	PhysicalAddress      [32]byte
	PermanentPhysicalAddress [32]byte
	Mtu                  uint32
	Type                 uint32
	TunnelType           uint32
	MediaType            uint32
	PhysicalMediumType   uint32
	AccessType           uint32
	DirectionType        uint32
	InterfaceAndOperStatusFlags uint8
	OperStatus           uint32
	AdminStatus          uint32
	MediaConnectState    uint32
	NetworkGuid          [16]byte
	ConnectionType       uint32
	_pad0                uint32 // alignment for the 64-bit counters below
	TransmitLinkSpeed    uint64
	ReceiveLinkSpeed     uint64
	InOctets             uint64
	InUcastPkts          uint64
	InNUcastPkts         uint64
	InDiscards           uint64
	InErrors             uint64
	InUnknownProtos      uint64
	InUcastOctets        uint64
	InMulticastOctets    uint64
	InBroadcastOctets    uint64
	OutOctets            uint64
	OutUcastPkts         uint64
	OutNUcastPkts        uint64
	OutDiscards          uint64
	OutErrors            uint64
	OutUcastOctets       uint64
	OutMulticastOctets   uint64
	OutBroadcastOctets   uint64
}

// mibIfTable2 is a variable-length table; NumEntries is followed by that many
// mibIfRow2 records. We index into it manually via pointer arithmetic.
type mibIfTable2 struct {
	NumEntries uint32
	_pad       uint32
	Table      [1]mibIfRow2
}

// adapterBytes returns the received and sent octets for the WinTun adapter,
// matched by its alias. Returns (0,0,false) if the adapter is not present.
func adapterBytes() (rx, tx uint64, ok bool) {
	var table *mibIfTable2
	ret, _, _ := procGetIfTable2.Call(uintptr(unsafe.Pointer(&table)))
	if ret != 0 || table == nil {
		return 0, 0, false
	}
	defer procFreeMibTable.Call(uintptr(unsafe.Pointer(table)))

	rows := unsafe.Slice(&table.Table[0], int(table.NumEntries))
	for i := range rows {
		if windows.UTF16ToString(rows[i].Alias[:]) == tunAdapterName {
			return rows[i].InOctets, rows[i].OutOctets, true
		}
	}
	return 0, 0, false
}

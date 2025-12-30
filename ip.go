package microps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/bugph0bia/go-microps/internal/util"
)

// ----------------------------------------------------------------------------
// 定数
// ----------------------------------------------------------------------------

const IPVersionIPV4 = 4

const IPHdrSizeMin = 20
const IPHdrSizeMax = 20

const IPTotalSizeMax = math.MaxUint16
const IPPayloadSizeMax = (IPTotalSizeMax - IPHdrSizeMax)

const IPAddrLen = 4

const (
	IPHdrFlagMF uint16 = 0x2000 // more flagments flag
	IPHdrFlagDF uint16 = 0x4000 // don't flagment flag
	IPHdrFlagRF uint16 = 0x8000 // reserved
)

const IPHdrOffsetMask uint16 = 0x1fff

const IPAddrAny IPAddr = 0x00000000       // 0.0.0.0
const IPAddrBroadcast IPAddr = 0xffffffff // 255.255.255.255

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

// IPアドレス型
type IPAddr uint32

func (ip IPAddr) String() string {
	addrs := make([]uint8, IPAddrLen)
	binary.NativeEndian.PutUint32(addrs, uint32(ip))
	return fmt.Sprintf("%d.%d.%d.%d", addrs[0], addrs[1], addrs[2], addrs[3])
}

func ParseIPAddr(str string) (IPAddr, error) {
	nums := strings.Split(str, ".")
	if len(nums) != IPAddrLen {
		return 0, fmt.Errorf("IP Address Parse failure")
	}

	addrs := make([]uint8, IPAddrLen)
	for i, num := range nums {
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, fmt.Errorf("IP Address Parse failure")
		}
		if n < 0 || 255 < n {
			return 0, fmt.Errorf("IP Address Parse failure")
		}
		addrs[i] = uint8(n)
	}

	ret := binary.NativeEndian.Uint32(addrs)
	return IPAddr(ret), nil
}

// IPヘッダ
type IPHdr struct {
	VHL      uint8  // Version & Header Length
	TOS      uint8  // Type Of Service
	Total    uint16 // Total Length
	ID       uint16 // Identification
	Offset   uint16 // Flags & Fragment Offset
	TTL      uint8  // Time To Live
	Protocol uint8  // Protocol
	Sum      uint16 // Header Checksum
	Src      IPAddr // Source Address
	Dst      IPAddr // Destination Address
}

// IPインタフェース
type IPIface struct {
	NetIfaceInfo
	unicast   IPAddr
	netmask   IPAddr
	broadcast IPAddr
}

func (iface *IPIface) Info() *NetIfaceInfo {
	return &iface.NetIfaceInfo
}

// IPプロトコル
type IPProtocol struct {
	NetProtocolInfo
}

func (proto *IPProtocol) Info() *NetProtocolInfo {
	return &proto.NetProtocolInfo
}

// 書籍では ip_input
func (proto *IPProtocol) InputHandler(data []uint8, dev NetDevice) {
	util.Debugf("dev=%s, len=%d", dev.Info().Name, len(data))

	// data を IPHdr に変換
	var hdr IPHdr
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.NativeEndian, &hdr)
	if err != nil {
		util.Errorf(err.Error())
		return
	}
	var v uint8 = hdr.VHL >> 4
	if v != IPVersionIPV4 {
		util.Errorf("ip version error: v=%d", v)
		return
	}
	var hlen uint8 = (hdr.VHL & 0x0f) << 2
	if len(data) < int(hlen) {
		util.Errorf("header length error: len=%d < hlen=%d", len(data), hlen)
		return
	}
	c, err := util.Cksum16(hdr, uint16(hlen), 0)
	if c != 0 || err != nil {
		util.Errorf("checksum error")
		return
	}
	total := util.Ntoh16(hdr.Total)
	if len(data) < int(total) {
		util.Errorf("total length error: len=%d < total=%d", len(data), total)
		return
	}
	offset := util.Ntoh16(hdr.Offset)
	if offset&IPHdrFlagMF > 0 || offset&IPHdrOffsetMask > 0 {
		util.Errorf("fragments does not support")
		return
	}

	i := NetDeviceGetIface(dev, NetIfaceFamilyIP)
	iface, ok := i.(*IPIface)
	if !ok {
		// 取得失敗したら何もしない
		return
	}
	if hdr.Dst != iface.unicast {
		if hdr.Dst != iface.broadcast && hdr.Dst != IPAddrBroadcast {
			// 別のホストへの通信のため無視
			return
		}
	}
	util.Debugf("permit, dev=%s, iface=%s", dev.Info().Name, iface.unicast.String())
	ipPrint(data[:total])
}

// NOTE: NetRun() を呼び出した後にエントリを追加/削除する場合はデバイスリストをロックすること
var ifaces []*IPIface

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

func IPIfaceAlloc(unicast string, netmask string) *IPIface {
	var iface IPIface
	iface.Info().family = NetIfaceFamilyIP

	var err error
	iface.unicast, err = ParseIPAddr(unicast)
	if err != nil {
		util.Errorf("ParseIPAddr() failure, addr=%s", unicast)
		return nil
	}
	iface.netmask, err = ParseIPAddr(netmask)
	if err != nil {
		util.Errorf("ParseIPAddr() failure, addr=%s", netmask)
		return nil
	}
	iface.broadcast = (iface.unicast & iface.netmask) | ^iface.netmask
	return &iface
}

// NOTE: NetRun() より前に呼び出すこと
func IPIfaceRegister(dev NetDevice, iface *IPIface) bool {
	util.Infof("dev=%s, %s, %s, %s", dev.Info().Name,
		iface.unicast.String(), iface.netmask.String(), iface.broadcast.String())
	if !NetDeviceAddIface(dev, iface) {
		util.Errorf("NetDeviceAddIntrerface() failure")
		return false
	}
	ifaces = append(ifaces, iface)
	return true
}

func IPIfaceSelect(addr IPAddr) *IPIface {
	for _, entry := range ifaces {
		if entry.unicast == addr {
			return entry
		}
	}
	return nil
}

func ipPrint(data []uint8) {
	// data を IPHdr に変換
	var hdr IPHdr
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.NativeEndian, &hdr)
	if err != nil {
		util.Errorf(err.Error())
		return
	}
	var v uint8 = hdr.VHL >> 4
	var hl uint8 = hdr.VHL & 0x0f
	var hlen uint8 = hl << 2

	var sb strings.Builder
	fmt.Fprintf(&sb, "        vhl: 0x%02x [v: %d, hl: %d (%d)]\n", hdr.VHL, v, hl, hlen)
	fmt.Fprintf(&sb, "        tos: 0x%02x\n", hdr.TOS)
	total := util.Ntoh16(hdr.Total)
	fmt.Fprintf(&sb, "      total: %d (payload: %d)\n", total, int(total)-int(hlen))
	fmt.Fprintf(&sb, "         id: %d\n", util.Ntoh16(hdr.ID))
	offset := util.Ntoh16(hdr.Offset)
	fmt.Fprintf(&sb, "     offset: 0x%04x [flags=%x, offset=%d]\n", offset, (offset >> 13), (offset & IPHdrOffsetMask))
	fmt.Fprintf(&sb, "        ttl: %d\n", hdr.TTL)
	fmt.Fprintf(&sb, "   protocol: %d\n", hdr.Protocol)
	fmt.Fprintf(&sb, "        sum: 0x%04x\n", util.Ntoh16(hdr.Sum))
	fmt.Fprintf(&sb, "        src: %s\n", hdr.Src.String())
	fmt.Fprintf(&sb, "        dst: %s\n", hdr.Dst.String())

	util.DebugDump(data)
	fmt.Fprintf(os.Stderr, sb.String())
}

func ipInit() bool {
	proto := IPProtocol{
		NetProtocolInfo{
			Typ: NetProtocolTypeIP,
		},
	}
	if !NetProtocolRegister(&proto) {
		util.Errorf("NetProtocolRegister() failure")
		return false
	}
	return true
}

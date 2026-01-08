package microps

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
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

// IP上位プロトコル種別
type IPUpperProtocolType uint8

const (
	IPUpperProtocolTypeICMP IPUpperProtocolType = 1
	IPUpperProtocolTypeTCP  IPUpperProtocolType = 6
	IPUpperProtocolTypeUDP  IPUpperProtocolType = 17
)

// ----------------------------------------------------------------------------
// インタフェース
// ----------------------------------------------------------------------------

// IP上位プロトコル
type IPUpperProtocol interface { // 書籍では ip_protocol
	Info() *IPUpperProtocolInfo
	InputHandler(ipHdr *IPHdr, data []uint8, ipIface *IPIface)
}

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

func ParseIPAddr(str string) (IPAddr, bool) {
	nums := strings.Split(str, ".")
	if len(nums) != IPAddrLen {
		return 0, false
	}

	addrs := make([]uint8, IPAddrLen)
	for i, num := range nums {
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, false
		}
		if n < 0 || 255 < n {
			return 0, false
		}
		addrs[i] = uint8(n)
	}

	ret := binary.NativeEndian.Uint32(addrs)
	return IPAddr(ret), true
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

// 書籍では ip_output_device()
func (iface *IPIface) Output(data []uint8, target IPAddr) bool {
	util.Debugf("dev=%s, len=%d, target=%s", iface.Info().Dev.Info().Name, len(data), target.String())

	var hwaddr [netDeviceAddrLen]uint8
	if iface.Info().Dev.Info().Flags&NetDeviceFlagNeedARP > 0 {
		if (target == iface.broadcast) || (target == IPAddrBroadcast) {
			hwaddr = iface.Dev.Info().Broadcast
		} else {
			util.Errorf("ARP does not implement")
			return false
		}
	}
	return NetDeviceOutput(iface.Info().Dev, NetProtocolTypeIP, data, hwaddr)
}

// IPプロトコル
type IPProtocol struct {
	NetProtocolInfo
}

func (proto *IPProtocol) Info() *NetProtocolInfo {
	return &proto.NetProtocolInfo
}

// 書籍では ip_input()
func (proto *IPProtocol) InputHandler(data []uint8, dev NetDevice) {
	util.Debugf("dev=%s, len=%d", dev.Info().Name, len(data))

	// data を IPHdr に変換
	var hdr IPHdr
	if !util.FromBytes(data, &hdr) {
		util.Errorf("FromBytes() failure")
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

	c, ok := util.Cksum16(hdr, int(hlen), 0)
	if !ok || c != 0 {
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
	IPPrint(data[:total])

	for _, upperProtocol := range upperProtocols {
		if upperProtocol.Info().Protocol == IPUpperProtocolType(hdr.Protocol) {
			upperProtocol.InputHandler(&hdr, data[hlen:], iface)
			return
		}
	}

	// サポート外のプロトコル
	if int(hlen+8) <= int(total) {
		// ICMPメッセージの応答として送信されるべきではない
		// ただし、ICMPは登録済みでここに到達することはない
		ICMPOutput(ICMPTypeDestUnreach, ICMPCodeProtoUnreach, 0, data[:hlen+8], iface.unicast, hdr.Src)
	}
}

// IP上位プロトコル情報
type IPUpperProtocolInfo struct {
	Protocol IPUpperProtocolType
}

// NOTE: NetRun() を呼び出した後にエントリを追加/削除する場合はデバイスリストをロックすること
var ifaces []*IPIface
var upperProtocols []IPUpperProtocol

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

func IPIfaceAlloc(unicast string, netmask string) *IPIface {
	var iface IPIface
	iface.Info().Family = NetIfaceFamilyIP

	var ok bool

	iface.unicast, ok = ParseIPAddr(unicast)
	if !ok {
		util.Errorf("ParseIPAddr() failure, addr=%s", unicast)
		return nil
	}

	iface.netmask, ok = ParseIPAddr(netmask)
	if !ok {
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

// NOTE: NetRun() より前に呼び出すこと
func IPUpperProtocolRegister(upperProtocol IPUpperProtocol) bool {
	for _, entry := range upperProtocols {
		if entry.Info().Protocol == upperProtocol.Info().Protocol {
			util.Errorf("already exists, protocol=%d", upperProtocol.Info().Protocol)
			return false
		}
	}

	upperProtocols = append(upperProtocols, upperProtocol)

	util.Infof("success, protocol=%d", upperProtocol.Info().Protocol)
	return true
}

func IPPrint(data []uint8) {
	// data を IPHdr に変換
	var hdr IPHdr
	if !util.FromBytes(data, &hdr) {
		util.Errorf("FromBytes() failure")
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

func IPBuildPacket(protocol IPUpperProtocolType, data []uint8, id uint16, offset uint16, src IPAddr, dst IPAddr) ([]uint8, bool) {
	var hlen uint16 = IPHdrSizeMin
	var total uint16 = hlen + uint16(len(data))

	var hdr IPHdr
	hdr.VHL = uint8((IPVersionIPV4 << 4) | (hlen >> 2))
	hdr.TOS = 0
	hdr.Total = util.Hton16(total)
	hdr.ID = util.Hton16(id)
	hdr.Offset = util.Hton16(offset)
	hdr.TTL = 0xff
	hdr.Protocol = uint8(protocol)
	hdr.Sum = 0
	hdr.Src = src
	hdr.Dst = dst

	hdr.Sum, _ = util.Cksum16(hdr, int(hlen), 0) // チェックサム値のバイトオーダー変換は行わない

	buf, ok := util.ToBytes(hdr)
	if !ok {
		util.Errorf("ToBytes() failure")
		return nil, false
	}
	buf = append(buf, data...)

	IPPrint(buf)
	return buf, true
}

func IPOutput(protocol IPUpperProtocolType, data []uint8, src IPAddr, dst IPAddr) (int, bool) {
	util.Debugf("%s => %s, protocol=%d, len=%d", src.String(), dst.String(), protocol, len(data))

	if src == IPAddrAny {
		util.Errorf("ip routing does not implement")
		return 0, false
	}

	iface := IPIfaceSelect(src)
	if iface == nil {
		util.Errorf("iface not found, src=%s", src.String())
		return 0, false
	}

	if ((dst & iface.netmask) != (iface.unicast & iface.netmask)) && (dst != IPAddrBroadcast) {
		util.Errorf("not reached, dst=%s", dst.String())
		return 0, false
	}

	if iface.Info().Dev.Info().MTU < IPHdrSizeMin+len(data) {
		util.Errorf("too long, dev=%s, mtu=%d < %d", iface.Info().Dev.Info().Name, iface.Info().Dev.Info().MTU, (IPHdrSizeMin + len(data)))
		return 0, false
	}

	id := rand.N[uint16](math.MaxUint16)
	buf, ok := IPBuildPacket(protocol, data, id, 0, iface.unicast, dst)
	if !ok {
		util.Errorf("IPBuildPacket() failure")
		return 0, false
	}

	if !iface.Output(buf, dst) {
		util.Errorf("iface.Output() failure")
		return 0, false
	}

	return len(buf), true
}

func IPInit() bool {
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

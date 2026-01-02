package microps

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bugph0bia/go-microps/internal/util"
)

// ----------------------------------------------------------------------------
// 定数
// ----------------------------------------------------------------------------

const EtherAddrLen = 6

const EtherHdrSize = 14
const EtherFrameSizeMin = 60   // FCS は除外
const EtherFrameSizeMax = 1514 // FCS は除外
const EtherPayloadSizeMin = (EtherFrameSizeMin - EtherHdrSize)
const EtherPayloadSizeMax = (EtherFrameSizeMax - EtherHdrSize)

// Ethernet種別
type EtherType uint16

const (
	EtherTypeIP   EtherType = 0x0800
	EtherTypeARP  EtherType = 0x0806
	EtherTypeIPV6 EtherType = 0x86dd
)

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

// Ethernetアドレス型
type EtherAddr [EtherAddrLen]uint8

func (ether EtherAddr) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", ether[0], ether[1], ether[2], ether[3], ether[4], ether[5])
}

func ParseEtherAddr(str string) (EtherAddr, bool) {
	hexs := strings.Split(str, ":")
	if len(hexs) != EtherAddrLen {
		return EtherAddr{}, false
	}

	var addrs EtherAddr
	for i, hex := range hexs {
		n, err := strconv.ParseUint(hex, 16, 8)
		if err != nil {
			return EtherAddr{}, false
		}
		if 255 < n {
			return EtherAddr{}, false
		}
		addrs[i] = uint8(n)
	}

	return addrs, true
}

var EtherAddrEmpty = EtherAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var EtherAddrBroadcast = EtherAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

// Ethernetヘッダ
type EtherHdr struct {
	Dst EtherAddr
	Src EtherAddr
	Typ EtherType
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

func EtherPrint(frame []uint8) {
	var hdr EtherHdr
	if !util.FromBytes(frame, &hdr) {
		util.Errorf("FromBytes() falure")
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "        src: %s\n", hdr.Src.String())
	fmt.Fprintf(&sb, "        dst: %s\n", hdr.Dst.String())
	fmt.Fprintf(&sb, "       type: 0x%04x\n", util.Ntoh16(uint16(hdr.Typ)))
	fmt.Fprintf(os.Stderr, sb.String())

	util.DebugDump(frame)
}

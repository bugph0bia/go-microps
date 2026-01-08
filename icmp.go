package microps

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/bugph0bia/go-microps/internal/util"
)

// ----------------------------------------------------------------------------
// 定数
// ----------------------------------------------------------------------------

const ICMPBufSize = IPPayloadSizeMax

// ICMPメッセージ種別
type ICMPType uint8

const (
	ICMPTypeEchoReply      ICMPType = 0
	ICMPTypeDestUnreach    ICMPType = 3
	ICMPTypeSourceQuench   ICMPType = 4
	ICMPTypeRedirect       ICMPType = 5
	ICMPTypeEcho           ICMPType = 8
	ICMPTypeTimeExceeded   ICMPType = 11
	ICMPTypeParamProblem   ICMPType = 12
	ICMPTypeTimestamp      ICMPType = 13
	ICMPTypeTimestampReply ICMPType = 14
	ICMPTypeInfoRequest    ICMPType = 15
	ICMPTypeInfoReply      ICMPType = 16
)

var icmpTypeStrings = map[ICMPType]string{
	ICMPTypeEchoReply:      "EchoReply",
	ICMPTypeDestUnreach:    "DestinationUnreachable",
	ICMPTypeSourceQuench:   "SourceQuench",
	ICMPTypeRedirect:       "Redirect",
	ICMPTypeEcho:           "Echo",
	ICMPTypeTimeExceeded:   "TimeExceeded",
	ICMPTypeParamProblem:   "ParameterProblem",
	ICMPTypeTimestamp:      "Timestamp",
	ICMPTypeTimestampReply: "TimestampReply",
	ICMPTypeInfoRequest:    "InfomationRequest",
	ICMPTypeInfoReply:      "InfomationReply",
}

// ICMPコード種別
type ICMPCode uint8

const (
	ICMPCodeNetUnreach ICMPCode = iota
	ICMPCodeHostUnreach
	ICMPCodeProtoUnreach
	ICMPCodePortUnreach
	ICMPCodeFragmentNeeded
	ICMPCodeSourceRouteFailed
)

func (typ ICMPType) String() string {
	if str, ok := icmpTypeStrings[typ]; ok {
		return str
	} else {
		return "Unknown"
	}
}

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

// ICMPプロトコル
type ICMPProtocol struct {
	IPUpperProtocolInfo
}

func (proto *ICMPProtocol) Info() *IPUpperProtocolInfo {
	return &proto.IPUpperProtocolInfo
}

// 書籍では icmp_input()
func (proto *ICMPProtocol) InputHandler(ipHdr *IPHdr, data []uint8, ipIface *IPIface) {
	hdrSize := int(unsafe.Sizeof(ICMPHdr{}))
	if len(data) < hdrSize {
		util.Errorf("too short")
		return
	}

	c, ok := util.Cksum16(data, len(data), 0)
	if !ok || c != 0 {
		util.Errorf("checksum error")
		return
	}

	util.Debugf("%s => %s, len=%d", ipHdr.Src.String(), ipHdr.Dst.String(), len(data))
	util.DebugDump(data)
	ICMPPrint(data)

	var hdr ICMPHdr
	if !util.FromBytes(data, &hdr) {
		util.Errorf("FromBytes() failure")
		return
	}
	switch hdr.Typ {
	case ICMPTypeEcho:
		// 受信したインタフェースのアドレスを含めた応答
		ICMPOutput(ICMPTypeEchoReply, hdr.Code, hdr.Dep, data[hdrSize:], ipIface.unicast, ipHdr.Src)
	default:
		// 無視
	}
}

// ICMPヘッダ（共通フィールド）
type ICMPCommon struct {
	Typ  ICMPType
	Code ICMPCode
	Sum  uint16
}

// ICMPヘッダ
type ICMPHdr struct {
	ICMPCommon
	Dep uint32 // message dependent field
}

// ICMPヘッダ（Echo / Echo Reply）
type ICMPEcho struct {
	ICMPCommon
	ID  uint16
	Seq uint16
}

// ICMPヘッダ（Destination Unreacheble）
type ICMPDestUnreach struct {
	ICMPCommon
	Unused uint32
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

func ICMPPrint(data []uint8) {
	// data を IPHdr に変換
	var hdr ICMPHdr
	if !util.FromBytes(data, &hdr) {
		util.Errorf("FromBytes() failure")
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "       type: %d (%s)\n", hdr.Typ, hdr.Typ.String())
	fmt.Fprintf(&sb, "       code; %d\n", hdr.Code)
	fmt.Fprintf(&sb, "        sum: 0x%04x\n", util.Ntoh16(hdr.Sum))

	switch hdr.Typ {
	case ICMPTypeEchoReply:
		fallthrough

	case ICMPTypeEcho:
		var echo ICMPEcho
		if !util.FromBytes(data, &echo) {
			util.Errorf("FromBytes() falure")
			return
		}
		fmt.Fprintf(&sb, "         id: %d\n", util.Ntoh16(echo.ID))
		fmt.Fprintf(&sb, "        seq: %d\n", util.Ntoh16(echo.Seq))

	case ICMPTypeDestUnreach:
		var unreach ICMPDestUnreach
		if !util.FromBytes(data, &unreach) {
			util.Errorf("FromBytes() falure")
			return
		}
		fmt.Fprintf(&sb, "     unused: %d\n", util.Ntoh16(uint16(unreach.Unused)))

	default:
		fmt.Fprintf(&sb, "        dep: 0x%08x\n", util.Ntoh16(uint16(hdr.Dep)))
	}

	util.DebugDump(data)
	fmt.Fprintf(os.Stderr, sb.String())
}

func ICMPOutput(typ ICMPType, code ICMPCode, val uint32, data []uint8, src IPAddr, dst IPAddr) bool {
	hdr := ICMPHdr{
		ICMPCommon: ICMPCommon{
			Typ:  typ,
			Code: code,
			Sum:  0,
		},
		Dep: val,
	}

	if ICMPBufSize < int(unsafe.Sizeof(hdr))+len(data) {
		util.Errorf("too large")
		return false
	}

	// データ構築→チェックサム計算して格納→データ再構築
	var buf []uint8
	var ok bool
	buf, ok = util.ToBytes(hdr)
	if !ok {
		return false
	}
	buf = append(buf, data...)
	hdr.Sum, ok = util.Cksum16(buf, len(buf), 0)
	if !ok {
		return false
	}
	buf, ok = util.ToBytes(hdr)
	if !ok {
		return false
	}
	buf = append(buf, data...)

	util.Debugf("%s => %s, len=%d", src.String(), dst.String(), len(buf))
	ICMPPrint(buf)

	_, ok = IPOutput(IPUpperProtocolTypeICMP, buf, src, dst)
	return ok
}

func ICMPInit() bool {
	if !IPUpperProtocolRegister(&ICMPProtocol{
		IPUpperProtocolInfo{
			Protocol: IPUpperProtocolTypeICMP,
		},
	}) {
		util.Errorf("IPUpperProtocolRegister() failure")
		return false
	}

	return true
}

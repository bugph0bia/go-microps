package microps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/bugph0bia/go-microps/internal/util"
)

// ----------------------------------------------------------------------------
// 定数
// ----------------------------------------------------------------------------

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
	if len(data) < int(unsafe.Sizeof(ICMPHdr{})) {
		util.Errorf("too short")
		return
	}

	c, err := util.Cksum16(data, uint16(len(data)), 0)
	if c != 0 || err != nil {
		util.Errorf("checksum error")
		return
	}

	util.Debugf("%s => %s, len=%d", ipHdr.Src.String(), ipHdr.Dst.String(), len(data))
	util.DebugDump(data)
	icmpPrint(data)
}

// ICMPヘッダ（共通フィールド）
type ICMPCommon struct {
	Typ  ICMPType
	Code uint8
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

func icmpPrint(data []uint8) {
	// data を IPHdr に変換
	var hdr ICMPHdr
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.NativeEndian, &hdr)
	if err != nil {
		util.Errorf(err.Error())
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "        type; %d (%s)\n", hdr.Typ, hdr.Typ.String())
	fmt.Fprintf(&sb, "        code; %d\n", hdr.Code)
	fmt.Fprintf(&sb, "         sum: 0x%04x\n", util.Ntoh16(hdr.Sum))

	switch hdr.Typ {
	case ICMPTypeEchoReply:
		fallthrough

	case ICMPTypeEcho:
		var echo ICMPEcho
		reader := bytes.NewReader(data)
		err := binary.Read(reader, binary.NativeEndian, &echo)
		if err != nil {
			util.Errorf(err.Error())
			return
		}
		fmt.Fprintf(&sb, "          id: %d\n", util.Ntoh16(echo.ID))
		fmt.Fprintf(&sb, "         seq: %d\n", util.Ntoh16(echo.Seq))

	case ICMPTypeDestUnreach:
		var unreach ICMPDestUnreach
		reader := bytes.NewReader(data)
		err := binary.Read(reader, binary.NativeEndian, &unreach)
		if err != nil {
			util.Errorf(err.Error())
			return
		}
		fmt.Fprintf(&sb, "      unused: %d\n", util.Ntoh16(uint16(unreach.Unused)))

	default:
		fmt.Fprintf(&sb, "         dep: 0x%08x\n", util.Ntoh16(uint16(hdr.Dep)))
	}

	util.DebugDump(data)
	fmt.Fprintf(os.Stderr, sb.String())
}

func icmpInit() bool {
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

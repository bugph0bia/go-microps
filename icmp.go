package microps

import "github.com/bugph0bia/go-microps/internal/util"

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
	util.Debugf("%s => %s, len=%d", ipHdr.Src.String(), ipHdr.Dst.String(), len(data))
	util.DebugDump(data)
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

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

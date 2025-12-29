package microps

import "github.com/bugph0bia/go-microps/internal/util"

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

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
	util.DebugDump(data)
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

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

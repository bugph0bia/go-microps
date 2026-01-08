package microps

import (
	"math"

	"github.com/bugph0bia/go-microps/internal/util"
)

const loopbackMTU = math.MaxUint16 // IPダイアグラムの最大値

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

// ループバックデバイス
type LoopbackDevice struct {
	NetDeviceInfo
}

func (dev *LoopbackDevice) Info() *NetDeviceInfo {
	return &dev.NetDeviceInfo
}

func (dev *LoopbackDevice) Open() bool {
	// 実装なし
	return true
}

func (dev *LoopbackDevice) Close() bool {
	// 実装なし
	return true
}

func (dev *LoopbackDevice) Output(typ NetProtocolType, data []uint8, dst any) bool {
	util.Debugf("dv=%s, type=0x%04x, len=%d", dev.Name, typ, len(data))
	util.DebugDump(data)
	return NetInput(typ, data, dev)
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

func LoopbackInit() NetDevice {
	dev := LoopbackDevice{
		NetDeviceInfo{
			Typ:   NetDeviceTypeLoopback,
			MTU:   loopbackMTU,
			Flags: NetDeviceFlagLoopback,
			Hlen:  0, // non header
			Alen:  0, // non address
		},
	}
	if !NetDeviceRegister(&dev) {
		util.Errorf("NetDeviceRegister() failure")
		return nil
	}
	util.Infof("success, dev=%s", dev.Info().Name)
	return &dev
}

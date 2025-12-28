package driver

import (
	"math"

	"github.com/bugph0bia/go-microps"
	"github.com/bugph0bia/go-microps/internal/util"
)

const loopbackMTU = math.MaxUint16 // IPダイアグラムの最大値

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

// ループバックデバイス
type LoopbackDevice struct {
	microps.NetDeviceInfo
}

func (dev *LoopbackDevice) Info() *microps.NetDeviceInfo {
	// 実装なし
	return &dev.NetDeviceInfo
}

func (devl *LoopbackDevice) Open() bool {
	// 実装なし
	return true
}

func (dev *LoopbackDevice) Close() bool {
	// 実装なし
	return true
}

func (dev *LoopbackDevice) Output(typ microps.NetDeviceType, data []uint8, dst any) bool {
	util.Debugf("dv=%s, type=0x%04x, len=%d", dev.Name, typ, len(data))
	util.DebugDump(data)
	return microps.NetInput(typ, data, dev)
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

func LoopbackInit() bool {
	dev := LoopbackDevice{
		microps.NetDeviceInfo{
			Typ:   microps.NetDeviceTypeLoopback,
			MTU:   loopbackMTU,
			Flags: microps.NetDeviceFlagLoopback,
			Hlen:  0, // non header
			Alen:  0, // non address
		},
	}
	if !microps.NetDeviceRegister(&dev) {
		util.Errorf("NetDeviceRegister() failure")
		return false
	}
	util.Infof("success, dev=%s", dev.Info().Name)
	return true
}

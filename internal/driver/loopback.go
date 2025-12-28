package driver

import (
	"math"

	"github.com/bugph0bia/go-microps"
	"github.com/bugph0bia/go-microps/internal/util"
)

const loopbackMTU = math.MaxUint16 // IPダイアグラムの最大値

// ----------------------------------------------------------------------------
// ネットデバイス構造体（ループバック）
// ----------------------------------------------------------------------------

type LoopbackOps struct {
}

func (l LoopbackOps) Open(dev *microps.NetDevice) bool {
	// 実装なし
	return true
}

func (l LoopbackOps) Close(dev *microps.NetDevice) bool {
	// 実装なし
	return true
}

func (l LoopbackOps) Output(dev *microps.NetDevice, typ microps.NetDeviceType, data []uint8, dst any) bool {
	util.Debugf("dv=%s, type=0x%04x, len=%d", dev.Name, typ, len(data))
	util.DebugDump(data)
	return microps.NetInput(typ, data, dev)
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

func LoopbackInit() (bool, *microps.NetDevice) {
	ok, dev := microps.NetDeviceRegister(&microps.NetDevice{
		Typ:   microps.NetDeviceTypeLoopback,
		MTU:   loopbackMTU,
		Flags: microps.NetDeviceFlagLoopback,
		Hlen:  0, // no header
		Alen:  0, // no address
		Ops:   LoopbackOps{},
	})
	if !ok {
		util.Errorf("NetDeviceRegister() failure")
		return false, nil
	}
	util.Infof("success, dev=%s", dev.Name)
	return true, dev
}

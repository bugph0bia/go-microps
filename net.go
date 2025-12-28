package microps

import (
	"fmt"

	"github.com/bugph0bia/go-microps/internal/util"
)

// ----------------------------------------------------------------------------
// 定数
// ----------------------------------------------------------------------------

// ネットデバイスの種別
type NetDeviceType uint16

const (
	NetDeviceTypeDummy NetDeviceType = iota
	NetDeviceTypeLoopback
	NetDeviceTypeEthernet
)

// ネットデバイスのフラグ
type NetDeviceFlag uint16

const (
	NetDeviceFlagUp        NetDeviceFlag = 0x0001
	NetDeviceFlagLoopback  NetDeviceFlag = 0x0010
	NetDeviceFlagBroadcast NetDeviceFlag = 0x0020
	NetDeviceFlagP2p       NetDeviceFlag = 0x0040
	NetDeviceFlagNeedARP   NetDeviceFlag = 0x0100
)

// ----------------------------------------------------------------------------
// インタフェース
// ----------------------------------------------------------------------------

// ネットデバイス
type NetDevice interface {
	Info() *NetDeviceInfo
	Open() bool  // 不要な場合は return true のみ実装する
	Close() bool // 不要な場合は return true のみ実装する
	Output(typ NetDeviceType, data []uint8, dst any) bool
}

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

const netDeviceAddrLen = 16

// ネットデバイス情報
type NetDeviceInfo struct {
	Name      string
	Typ       NetDeviceType
	MTU       int
	Flags     NetDeviceFlag
	Hlen      int
	Alen      int
	Addr      [netDeviceAddrLen]uint8
	Broadcast [netDeviceAddrLen]uint8
	Priv      any
}

func (dev NetDeviceInfo) IsUp() bool {
	return (dev.Flags & NetDeviceFlagUp) > 0x0000
}

func (dev NetDeviceInfo) State() string {
	if dev.IsUp() {
		return "UP"
	} else {
		return "DOWN"
	}
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

// NOTE: NetRun() を呼び出した後にエントリを追加/削除する場合はデバイスリストをロックすること

var devices []NetDevice

// NOTE: NetRun() より後に呼び出すこと
func NetDeviceRegister(dev NetDevice) bool {
	dev.Info().Name = fmt.Sprintf("net%d", len(devices)+1)
	devices = append(devices, dev)
	util.Infof("success, dev=%s, type=0x%04x", dev.Info().Name, dev.Info().Typ)
	return true
}

func NetDeviceOpen(dev NetDevice) bool {
	util.Infof("dev=%s", dev.Info().Name)
	if dev.Info().IsUp() {
		util.Errorf("already opened, dev=%s", dev.Info().Name)
		return false
	}
	if !dev.Open() {
		util.Errorf("failure, dev=%s", dev.Info().Name)
		return false
	}
	dev.Info().Flags |= NetDeviceFlagUp
	return true
}

func NetDeviceClose(dev NetDevice) bool {
	util.Infof("dev=%s", dev.Info().Name)
	if !dev.Info().IsUp() {
		util.Errorf("not opened, dev=%s", dev.Info().Name)
		return false
	}
	if !dev.Close() {
		util.Errorf("failure, dev=%s", dev.Info().Name)
		return false
	}
	dev.Info().Flags &^= NetDeviceFlagUp
	return true
}

func NetDeviceOutput(dev NetDevice, typ NetDeviceType, data []uint8, dst any) bool {
	util.Debugf("dev=%s, type=0x%04x, %d", dev.Info().Name, typ, len(data))
	util.DebugDump(data)
	if !dev.Info().IsUp() {
		util.Errorf("not opened, dev=%s", dev.Info().Name)
		return false
	}
	if dev.Info().MTU < len(data) {
		util.Errorf("too long, dev=%s, mtu=%d, len=%d", dev.Info().Name, dev.Info().MTU, len(data))
	}
	if !dev.Output(typ, data, dst) {
		util.Errorf("failure, dev=%s, mtu=%d, len=%d", dev.Info().Name, dev.Info().MTU, len(data))
		return false
	}
	return true
}

func NetInput(typ NetDeviceType, data []uint8, dev NetDevice) bool {
	util.Debugf("dev=%s, type=0x%04x, len=%d", dev.Info().Name, typ, len(data))
	util.DebugDump(data)
	return true
}

func NetInit() bool {
	util.Infof("initialize...")
	if !platformInit() {
		util.Errorf("platformInit() failuer")
		return false
	}
	util.Infof("success")
	return true
}

func NetRun() bool {
	util.Infof("startup...")
	if !platformRun() {
		util.Errorf("platformRun() failuer")
		return false
	}
	for i := range devices {
		NetDeviceOpen(devices[i])
	}
	util.Infof("success")
	return true
}

func NetShutdown() bool {
	util.Infof("shutting down...")
	if !platformShutdown() {
		util.Errorf("platformShutdown() failuer")
		return false
	}
	for i := range devices {
		NetDeviceClose(devices[i])
	}
	util.Infof("success")
	return true
}

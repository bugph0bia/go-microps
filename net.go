package microps

import (
	"fmt"

	"github.com/bugph0bia/go-microps/internal/util"
)

// ----------------------------------------------------------------------------
// ネットデバイスの種類
// ----------------------------------------------------------------------------
type NetDeviceType uint16

const (
	NetDeviceTypeDummy NetDeviceType = iota
	NetDeviceTypeLoopback
	NetDeviceTypeEthernet
)

// ----------------------------------------------------------------------------
// ネットデバイスのフラグ
// ----------------------------------------------------------------------------

type NetDeviceFlag uint16

const (
	NetDeviceFlagUp        NetDeviceFlag = 0x0001
	NetDeviceFlagLoopback  NetDeviceFlag = 0x0010
	NetDeviceFlagBroadcast NetDeviceFlag = 0x0020
	NetDeviceFlagP2p       NetDeviceFlag = 0x0040
	NetDeviceFlagNeedARP   NetDeviceFlag = 0x0100
)

// ----------------------------------------------------------------------------
// デバイスドライバのインタフェース
// ----------------------------------------------------------------------------

type NetDeviceOps interface {
	// Open, Close は任意、不要な場合は return true のみ実装する
	Open(dev *NetDevice) bool
	Close(dev *NetDevice) bool
	Output(dev *NetDevice, typ NetDeviceType, data []uint8, dst any) bool
}

// ----------------------------------------------------------------------------
// ネットデバイス構造体
// ----------------------------------------------------------------------------

const netDeviceAddrLen = 16

type NetDevice struct {
	Name      string
	Typ       NetDeviceType
	MTU       int
	Flags     NetDeviceFlag
	Hlen      int
	Alen      int
	Addr      [netDeviceAddrLen]uint8
	Broadcast [netDeviceAddrLen]uint8
	Ops       NetDeviceOps
	Priv      any
}

func (dev *NetDevice) IsUp() bool {
	return (dev.Flags & NetDeviceFlagUp) > 0x0000
}

func (dev *NetDevice) State() string {
	if dev.IsUp() {
		return "UP"
	} else {
		return "DOWN"
	}
}

func (dev *NetDevice) Open() bool {
	util.Infof("dev=%s", dev.Name)
	if dev.IsUp() {
		util.Errorf("already opened, dev=%s", dev.Name)
		return false
	}
	if !dev.Ops.Open(dev) {
		util.Errorf("failure, dev=%s", dev.Name)
		return false
	}
	dev.Flags |= NetDeviceFlagUp
	return true
}

func (dev *NetDevice) Close() bool {
	util.Infof("dev=%s", dev.Name)
	if !dev.IsUp() {
		util.Errorf("not opened, dev=%s", dev.Name)
		return false
	}
	if !dev.Ops.Close(dev) {
		util.Errorf("failure, dev=%s", dev.Name)
		return false
	}
	dev.Flags &^= NetDeviceFlagUp
	return true
}

func (dev *NetDevice) Output(typ NetDeviceType, data []uint8, dst any) bool {
	util.Debugf("dev=%s, type=0x%04x, %d", dev.Name, typ, len(data))
	util.DebugDump(data)
	if !dev.IsUp() {
		util.Errorf("not opened, dev=%s", dev.Name)
		return false
	}
	if dev.MTU < len(data) {
		util.Errorf("too long, dev=%s, mtu=%d, len=%d", dev.Name, dev.MTU, len(data))
	}
	if !dev.Ops.Output(dev, typ, data, dst) {
		util.Errorf("failure, dev=%s, mtu=%d, len=%d", dev.Name, dev.MTU, len(data))
		return false
	}
	return true
}

// ----------------------------------------------------------------------------
// ネットデバイスリスト
// ----------------------------------------------------------------------------

type NetDevices []NetDevice

func (devs NetDevices) Register(dev *NetDevice) bool {
	dev.Name = fmt.Sprintf("net%d", len(devices)+1)
	devices = append(devices, *dev)
	util.Infof("success, dev=%s, type=0x%04x", dev.Name, dev.Typ)
	return true
}

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

// NOTE: netRun() を呼び出した後にエントリを追加/削除する場合は、
//
//	デバイスリストをロックする必要がある
var devices NetDevices

func NetDeviceRegister(dev *NetDevice) (bool, *NetDevice) {
	if devices.Register(dev) {
		return true, &devices[len(devices)-1]
	} else {
		return false, nil
	}
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
		devices[i].Open()
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
		devices[i].Close()
	}
	util.Infof("success")
	return true
}

func NetInput(typ NetDeviceType, data []uint8, dev *NetDevice) bool {
	util.Debugf("dev=%s, type=0x%04x, len=%d", dev.Name, typ, len(data))
	util.DebugDump(data)
	return true
}

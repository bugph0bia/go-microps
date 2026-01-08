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

// ネットインタフェースの種別
type NetIfaceFamily uint16

const (
	NetIfaceFamilyIP NetIfaceFamily = iota
	NetIfaceFamilyIPV6
)

// ネットプロトコルの種別
type NetProtocolType uint16

const (
	NetProtocolTypeIP   NetProtocolType = 0x0800
	NetProtocolTypeARP  NetProtocolType = 0x0806
	NetProtocolTypeIPV6 NetProtocolType = 0x86dd
)

// ----------------------------------------------------------------------------
// インタフェース
// ----------------------------------------------------------------------------

// ネットデバイス
type NetDevice interface {
	Info() *NetDeviceInfo
	// Open, Close が不要な場合は return true のみ実装すること
	Open() bool
	Close() bool
	Output(typ NetProtocolType, data []uint8, dst any) bool
}

// ネットインタフェース
type NetIface interface {
	Info() *NetIfaceInfo
}

// ネットプロトコル
type NetProtocol interface {
	Info() *NetProtocolInfo
	InputHandler(data []uint8, dev NetDevice)
}

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

const netDeviceAddrLen = 16

// ネットデバイス情報
type NetDeviceInfo struct {
	ifaces    []NetIface
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

// ネットインタフェース情報
type NetIfaceInfo struct {
	Dev    NetDevice // 親への参照
	Family NetIfaceFamily
}

// ネットプロトコル情報
type NetProtocolInfo struct {
	Typ NetProtocolType
}

// NOTE: NetRun() を呼び出した後にエントリを追加/削除する場合はデバイスリストをロックすること
var Devices []NetDevice
var Protocols []NetProtocol

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

// NOTE: NetRun() より前に呼び出すこと
func NetDeviceRegister(dev NetDevice) bool {
	dev.Info().Name = fmt.Sprintf("net%d", len(Devices))
	Devices = append(Devices, dev)

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

func NetDeviceOutput(dev NetDevice, typ NetProtocolType, data []uint8, dst any) bool {
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

// NOTE: NetRun() より前に呼び出すこと
func NetDeviceAddIface(dev NetDevice, iface NetIface) bool {
	for _, entry := range dev.Info().ifaces {
		if entry.Info().Family == iface.Info().Family {
			// NOTE: 簡単のために、１つのファミリのインタフェースは１つのみ紐づけ可能とする
			util.Errorf("already exists, dev=%s, family=%d", dev.Info().Name, entry.Info().Family)
			return false
		}
	}

	dev.Info().ifaces = append(dev.Info().ifaces, iface)
	iface.Info().Dev = dev

	util.Infof("success, dev=%s", dev.Info().Name)
	return true
}

func NetDeviceGetIface(dev NetDevice, family NetIfaceFamily) NetIface {
	for _, entry := range dev.Info().ifaces {
		if entry.Info().Family == family {
			return entry
		}
	}
	return nil
}

// NOTE: NetRun() より前に呼び出すこと
func NetProtocolRegister(proto NetProtocol) bool {
	for _, p := range Protocols {
		if proto.Info().Typ == p.Info().Typ {
			util.Errorf("already registerd, type=0x%04d", p.Info().Typ)
			return false
		}
	}
	Protocols = append(Protocols, proto)

	util.Infof("success, type=0x%04x", proto.Info().Typ)
	return true
}

func NetInput(typ NetProtocolType, data []uint8, dev NetDevice) bool {
	util.Debugf("dev=%s, type=0x%04x, len=%d", dev.Info().Name, typ, len(data))
	util.DebugDump(data)

	for _, proto := range Protocols {
		if proto.Info().Typ == typ {
			proto.InputHandler(data, dev)
			return true
		}
	}

	// 未サポートのプロトコルの場合はここを通る
	return true
}

func NetInit() bool {
	util.Infof("initialize...")

	if !PlatformInit() {
		util.Errorf("platformInit() failure")
		return false
	}

	if !IPInit() {
		util.Errorf("ipInit() failure")
		return false
	}

	if !ICMPInit() {
		util.Errorf("icmpInit() failure")
		return false
	}

	util.Infof("success")
	return true
}

func NetRun() bool {
	util.Infof("startup...")

	if !PlatformRun() {
		util.Errorf("platformRun() failure")
		return false
	}

	for i := range Devices {
		NetDeviceOpen(Devices[i])
	}

	util.Infof("success")
	return true
}

func NetShutdown() bool {
	util.Infof("shutting down...")

	if !PlatformShutdown() {
		util.Errorf("platformShutdown() failure")
		return false
	}

	for i := range Devices {
		NetDeviceClose(Devices[i])
	}

	util.Infof("success")
	return true
}

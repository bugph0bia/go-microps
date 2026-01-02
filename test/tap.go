//go:build TAP

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/bugph0bia/go-microps"
	"github.com/bugph0bia/go-microps/internal/util"
)

// TAP デバイスファイル
const cloneDevice = "/dev/net/tun"

// システムコールに使用する定数
const (
	TUNSETIFF = 0x400454ca
	IFF_TAP   = 0x0002
	IFF_NO_PI = 0x1000
)

// ネットインタフェース設定用の構造体
type ifreq struct {
	ifrName  [16]byte
	ifrFlags int16
}

func main() {

	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <ifname>\n", os.Args[0])
		os.Exit(-1)
	}

	// デバイスをオープン
	file, err := os.OpenFile(cloneDevice, os.O_RDWR, 0)
	if err != nil {
		util.Errorf("open: %s", err.Error())
		os.Exit(-1)
	}
	defer file.Close()

	// TAPデバイスを取得
	var ifr ifreq
	ifname := os.Args[1]
	copy(ifr.ifrName[:], []uint8(ifname))
	ifr.ifrFlags = IFF_TAP | IFF_NO_PI
	_, _, sysErr := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr)))
	if sysErr != 0 {
		util.Errorf("ioctl [TUNSETIFF]: %s", sysErr.Error())
		os.Exit(-1)
	}

	util.Infof("waiting for packets from <%s>...", ifname)

	buf := make([]uint8, 2048)
	for true {
		n, err := file.Read(buf)
		if err != nil {
			util.Errorf("recv: %s", err.Error())
			os.Exit(-1)
		}
		util.Debugf("receive %d bytes via %s", n, ifname)
		microps.EtherPrint(buf[:n])
	}
	os.Exit(0)
}

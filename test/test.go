//go:build !TAP

package main

import (
	"context"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/bugph0bia/go-microps"
	"github.com/bugph0bia/go-microps/internal/util"
)

// ループバックアドレス
const loopbackIPAddr = "127.0.0.1"
const loopbackNetmask = "255.0.0.0"

// TAP
const etherTAPName = "tap0"
const etherTAPHWAddr = "00:00:5e:00:53:01"
const etherTAPIPAddr = "192.0.2.2"
const etherTAPNetmask = "255.255.255.0"

const defaultGateway = "192.0.2.1"

var testData = []uint8{
	0x45, 0x00, 0x00, 0x30,
	0x00, 0x80, 0x00, 0x00,
	0xff, 0x01, 0xbd, 0x4a,
	0x7f, 0x00, 0x00, 0x01,
	0x7f, 0x00, 0x00, 0x01,
	0x08, 0x00, 0x35, 0x64,
	0x00, 0x80, 0x00, 0x01,
	0x31, 0x32, 0x33, 0x34,
	0x35, 0x36, 0x37, 0x38,
	0x39, 0x30, 0x21, 0x40,
	0x23, 0x24, 0x25, 0x5e,
	0x26, 0x2a, 0x28, 0x29,
}

var terminate bool = false

func main() {
	ret := true

	// シグナルによる割り込み処理
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// onSignal
	go func() {
		<-ctx.Done()
		terminate = true
	}()

	if !setup() {
		util.Errorf("setup() failure")
		os.Exit(-1)
	}

	ret = appMain()

	if !cleanup() {
		util.Errorf("cleanup() failure")
		os.Exit(-1)
	}

	if ret {
		os.Exit(0)
	} else {
		os.Exit(-1)
	}
}

func setup() bool {
	util.Infof("setup protocol stack...")

	if !microps.NetInit() {
		util.Errorf("netInit() failure")
		return false
	}

	dev := microps.LoopbackInit()
	if dev == nil {
		util.Errorf("LoopbackInit() falure")
		return false
	}

	iface := microps.IPIfaceAlloc(loopbackIPAddr, loopbackNetmask)
	if iface == nil {
		util.Errorf("IPIfaceAlloc() failure")
		return false
	}
	if !microps.IPIfaceRegister(dev, iface) {
		util.Errorf("IPIfaceRegister() failure")
		return false
	}

	if !microps.NetRun() {
		util.Errorf("netRun() failure")
		return false
	}

	return true
}

func appMain() bool {
	src, _ := microps.ParseIPAddr(loopbackIPAddr)
	dst := src
	var id uint32 = uint32(os.Getpid() % math.MaxUint16)
	var seq uint32 = 0
	data := []uint8("TEST")

	util.Debugf("press Ctrl+C to terminate")

	for !terminate {
		seq++
		var val uint32 = util.Hton32(id<<16 | seq)
		if ok := microps.ICMPOutput(microps.ICMPTypeEcho, 0, val, data, src, dst); !ok {
			util.Errorf("ICMPOutput() failure")
			return false
		}
		time.Sleep(1 * time.Second)
	}

	util.Debugf("terminate")
	return true
}

func cleanup() bool {
	util.Infof("cleanup protocol stack...")

	if !microps.NetShutdown() {
		util.Errorf("NetShutdown() failure")
		return false
	}
	return true
}

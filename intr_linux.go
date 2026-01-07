package microps

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bugph0bia/go-microps/internal/util"
)

// ----------------------------------------------------------------------------
// 定数
// ----------------------------------------------------------------------------

const IntrIRQBase = syscall.SIGUSR1

const (
	IntrIRQFlagShared uint16 = 0x0001 // IRQ番号の共有を許可
)

// ----------------------------------------------------------------------------
// データ
// ----------------------------------------------------------------------------

// 割り込みハンドラ型
type IntrISRHandler func(sig syscall.Signal, dev NetDevice)

// 割り込み管理構造体
type IRQEntry struct {
	// 書籍ではIRQ番号を意味する irq
	sig syscall.Signal
	// 割り込みハンドラ (Interrupt Service Routine)
	isr IntrISRHandler
	// フラグ
	flags uint16
	// 割り込み元となるネットワークデバイス
	dev NetDevice
}

// NOTE: intrRun() を呼び出した後にエントリを追加/削除する場合はデバイスリストをロックすること
var irqs []IRQEntry

// シグナル受信用のチャネル
var sigChan = make(chan os.Signal, 1)

// シグナル受信ルーチンの制御用チャネル
var ready = make(chan struct{})     // 起動直後の同期
var terminate = make(chan struct{}) // 終了指示

// ----------------------------------------------------------------------------
// メインロジック
// ----------------------------------------------------------------------------

func intrRegister(sig syscall.Signal, isr IntrISRHandler, flags uint16, dev NetDevice) bool {
	util.Debugf("sig=%s, flags=0x%04x, dev=%s", sig.String(), flags, dev.Info().Name)

	for _, entry := range irqs {
		if entry.sig == sig {
			if (entry.flags&IntrIRQFlagShared == 0) || (flags&IntrIRQFlagShared == 0) {
				util.Errorf("conflicts with registerd IRQs")
				return false
			}
		}
	}

	entry := IRQEntry{
		sig:   sig,
		isr:   isr,
		flags: flags,
		dev:   dev,
	}
	irqs = append(irqs, entry)
	signal.Notify(sigChan, sig)

	util.Debugf("registerd: sig=%s, dev=%s", entry.sig.String(), entry.dev.Info().Name)
	return true
}

func intrInit() bool {
	signal.Stop(sigChan)
	return true
}

func intrRun() bool {
	go intrMain()
	<-ready // シグナル受信ルーチンの準備完了を待機
	return true
}

func intrShutdown() bool {
	close(terminate) // チャネルを閉じて終了指示
	return true
}

func intrMain() {
	util.Debugf("start...")
	close(ready) // チャネルを閉じて準備完了を通知

LOOP:
	for {
		select {
		// ルーチンの終了
		case <-terminate:
			break LOOP

		// シグナル受信
		case sig := <-sigChan:
			for _, entry := range irqs {
				if entry.sig == sig {
					util.Debugf("sig=%s, name=%s", entry.sig.String(), entry.dev.Info().Name)
					entry.isr(entry.sig, entry.dev)
					if (entry.flags ^ IntrIRQFlagShared) > 0 {
						break
					}
				}
			}
			break
		}
	}
	util.Debugf("terminated")
}

package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"
)

// ------------------------------------------------------------------------------------------------------
// ロギング
// ------------------------------------------------------------------------------------------------------

func Errorf(format string, a ...any) int {
	return logf('e', format, a...)
}

func Warnf(format string, a ...any) int {
	return logf('W', format, a...)
}

func Infof(format string, a ...any) int {
	return logf('I', format, a...)
}

func Debugf(format string, a ...any) int {
	return logf('D', format, a...)
}

func logf(level rune, format string, a ...any) int {
	var funcName string

	// ランタイム情報から、2つ上の呼び出し元の関数情報を取得する
	pc, fpath, line, ok := runtime.Caller(2) // この関数は直接呼び出さず、errorf などのラッパーを中継する前提
	if !ok {
		funcName = "Unknown"
		fpath = "Unknown"
		line = 0
	} else {
		// プログラムカウンタから パッケージ名.関数名 を取得して、関数名のみ抽出
		funcName = runtime.FuncForPC(pc).Name()
		fns := strings.Split(funcName, ".")
		funcName = fns[len(fns)-1]
	}

	return lprintf(os.Stderr, level, filepath.Base(fpath), line, funcName, format, a...)
}

var mtx sync.Mutex

func lprintf(w io.Writer, level rune, fileName string, line int, funcName string, format string, a ...any) int {
	mtx.Lock()
	defer mtx.Unlock()

	var sb strings.Builder
	t := time.Now()
	fmt.Fprintf(&sb, "%s [%c] %s: ", t.Format("15:04:05.000"), level, funcName)
	fmt.Fprintf(&sb, format, a...)
	fmt.Fprintf(&sb, " (%s:%d)\n", fileName, line)

	n, _ := fmt.Fprintf(w, sb.String())
	return n
}

func hexdump(w io.Writer, data any) {
	mtx.Lock()
	defer mtx.Unlock()

	// data を []byte に変換
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.NativeEndian, data)
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
	}
	src := buf.Bytes()

	var sb strings.Builder
	fmt.Fprintln(&sb, "+------+-------------------------------------------------+------------------+")
	for offset := 0; offset < len(src); offset += 16 {
		fmt.Fprintf(&sb, "| %04x | ", offset)
		for index := 0; index < 16; index++ {
			if offset+index < len(src) {
				fmt.Fprintf(&sb, "%02x ", 0xFF&src[offset+index])
			} else {
				fmt.Fprint(&sb, "   ")
			}
		}
		fmt.Fprintf(&sb, "| ")
		for index := 0; index < 16; index++ {
			if offset+index < len(src) {
				if src[offset+index] < unicode.MaxASCII && unicode.IsPrint(rune(src[offset+index])) {
					fmt.Fprintf(&sb, "%c", src[offset+index])
				} else {
					fmt.Fprint(&sb, ".")
				}
			} else {
				fmt.Fprint(&sb, " ")
			}
		}
		fmt.Fprintln(&sb, " |")
	}
	fmt.Fprintln(&sb, "+------+-------------------------------------------------+------------------+")

	fmt.Fprint(w, sb.String())
}

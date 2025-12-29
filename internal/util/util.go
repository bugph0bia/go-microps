package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unsafe"
)

// ----------------------------------------------------------------------------
// ロギング
// ----------------------------------------------------------------------------

func Errorf(format string, a ...any) int {
	return logf('E', format, a...)
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
		// プログラムカウンタから関数名を取得
		funcName = runtime.FuncForPC(pc).Name()
		// パッケージ名を除去
		re := regexp.MustCompile(`(.+/)+.+?\.`)
		funcName = re.ReplaceAllString(funcName, "")
	}

	return lprintf(os.Stderr, level, filepath.Base(fpath), line, funcName, format, a...)
}

func lprintf(w io.Writer, level rune, fileName string, line int, funcName string, format string, a ...any) int {
	var sb strings.Builder
	t := time.Now()
	fmt.Fprintf(&sb, "%s [%c] %s: ", t.Format("15:04:05.000"), level, funcName)
	fmt.Fprintf(&sb, format, a...)
	fmt.Fprintf(&sb, " (%s:%d)\n", fileName, line)

	n, _ := fmt.Fprintf(w, sb.String())
	return n
}

func hexdump(w io.Writer, data any) {
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

// ----------------------------------------------------------------------------
// バイトオーダー
// ----------------------------------------------------------------------------

const (
	BigEndian    = 4321
	LittleEndian = 1234
)

func ByteOrder() int {
	var x int32 = 1
	if *(*byte)(unsafe.Pointer(&x)) == 1 {
		return LittleEndian
	} else {
		return BigEndian
	}
}

func Hton16(h uint16) uint16 {
	if ByteOrder() == LittleEndian {
		return bits.ReverseBytes16(h)
	} else {
		return h
	}
}

func Ntoh16(n uint16) uint16 {
	if ByteOrder() == LittleEndian {
		return bits.ReverseBytes16(n)
	} else {
		return n
	}
}

func Hton32(h uint32) uint32 {
	if ByteOrder() == LittleEndian {
		return bits.ReverseBytes32(h)
	} else {
		return h
	}
}

func Ntoh32(n uint32) uint32 {
	if ByteOrder() == LittleEndian {
		return bits.ReverseBytes32(n)
	} else {
		return n
	}
}

// ----------------------------------------------------------------------------
// チェックサム
// ----------------------------------------------------------------------------

func Cksum16(data any, count uint16, init uint32) (uint16, error) {
	// data を []byte に変換
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.NativeEndian, data)
	if err != nil {
		Errorf(err.Error())
		return 0, err
	}
	b := buf.Bytes()
	// []byte を []uint16 に変換
	addr := make([]uint16, len(b)/2)
	for i := 0; i < len(addr); i++ {
		addr[i] = binary.NativeEndian.Uint16(b[i*2:])
	}

	var sum uint32 = init

	i := 0
	for count > 1 {
		sum += uint32(addr[i])
		i++
		count -= 2
	}
	if count > 0 {
		sum += uint32(addr[i] >> 8)
	}
	for (sum >> 16) > 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum), nil
}

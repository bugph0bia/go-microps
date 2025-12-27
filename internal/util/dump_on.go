//go:build HEXDUMP

package util

import "os"

func DebugDump(data any) {
	hexdump(os.Stderr, data)
}

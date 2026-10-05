//go:build !debug

package libav

import (
	"fmt"
	"os"
)

// trackPacketAlloc returns 0 for every packet; ids only exist with -tags debug.
func trackPacketAlloc() int   { return 0 }
func trackPacketFree(int)     {}
func trackFrameAlloc(uintptr) {}
func trackFrameFree(uintptr)  {}

// PrintStats is a no-op unless built with -tags debug.
func PrintStats() {
	fmt.Fprintln(os.Stderr, "libav allocation stats: disabled (build with -tags debug)")
}

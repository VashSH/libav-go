package libav

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// clanker generated code below do not trust!

// Allocation statistics for Packet and Frame. Every allocation records the
// call site (first frame outside this package) keyed by packet id / frame
// pointer; Free removes it. PrintStats dumps the counters and whatever is
// still live, grouped by call site, so leaks point at the code that made them.

type allocStats struct {
	packetsCreated     atomic.Uint64
	packetsFreed       atomic.Uint64
	packetsUnknownFree atomic.Uint64
	framesCreated      atomic.Uint64
	framesFreed        atomic.Uint64
	framesUnknownFree  atomic.Uint64

	mu          sync.Mutex
	livePackets map[int]string
	liveFrames  map[uintptr]string
}

var stats = allocStats{
	livePackets: map[int]string{},
	liveFrames:  map[uintptr]string{},
}

var libavPkgPrefix = func() string {
	name := runtime.FuncForPC(reflect.ValueOf(PrintStats).Pointer()).Name()
	return name[:strings.LastIndex(name, ".")+1]
}()

func callSite() string {
	var pcs [16]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	last := "unknown"
	for {
		fr, more := frames.Next()
		if fr.Function != "" {
			last = fmt.Sprintf("%s:%d (%s)", fr.File, fr.Line, fr.Function)
			if !strings.HasPrefix(fr.Function, libavPkgPrefix) {
				return last
			}
		}
		if !more {
			return last
		}
	}
}

func trackPacketAlloc(id int) {
	site := callSite()
	stats.mu.Lock()
	stats.livePackets[id] = site
	stats.mu.Unlock()
	stats.packetsCreated.Add(1)
}

func trackPacketFree(id int) {
	stats.mu.Lock()
	_, ok := stats.livePackets[id]
	delete(stats.livePackets, id)
	stats.mu.Unlock()
	if ok {
		stats.packetsFreed.Add(1)
	} else {
		stats.packetsUnknownFree.Add(1)
	}
}

func trackFrameAlloc(ptr uintptr) {
	site := callSite()
	stats.mu.Lock()
	stats.liveFrames[ptr] = site
	stats.mu.Unlock()
	stats.framesCreated.Add(1)
}

func trackFrameFree(ptr uintptr) {
	stats.mu.Lock()
	_, ok := stats.liveFrames[ptr]
	delete(stats.liveFrames, ptr)
	stats.mu.Unlock()
	if ok {
		stats.framesFreed.Add(1)
	} else {
		stats.framesUnknownFree.Add(1)
	}
}

type siteGroup struct {
	site string
	ids  []string
}

func groupBySite[K comparable](live map[K]string, format func(K) string) []siteGroup {
	bySite := map[string][]string{}
	for k, site := range live {
		bySite[site] = append(bySite[site], format(k))
	}
	groups := make([]siteGroup, 0, len(bySite))
	for site, ids := range bySite {
		sort.Strings(ids)
		groups = append(groups, siteGroup{site, ids})
	}
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].ids) != len(groups[j].ids) {
			return len(groups[i].ids) > len(groups[j].ids)
		}
		return groups[i].site < groups[j].site
	})
	return groups
}

func printGroups(label string, groups []siteGroup) {
	const maxIDs = 10
	for _, g := range groups {
		shown := g.ids
		more := ""
		if len(shown) > maxIDs {
			more = fmt.Sprintf(" ... +%d more", len(shown)-maxIDs)
			shown = shown[:maxIDs]
		}
		fmt.Fprintf(os.Stderr, "  %5d %s  %s\n        %ss: %s%s\n",
			len(g.ids), label, g.site, label, strings.Join(shown, ", "), more)
	}
}

// PrintStats writes the current Packet/Frame allocation counters and every
// still-live allocation grouped by call site to stderr.
func PrintStats() {
	stats.mu.Lock()
	livePackets := make(map[int]string, len(stats.livePackets))
	for k, v := range stats.livePackets {
		livePackets[k] = v
	}
	liveFrames := make(map[uintptr]string, len(stats.liveFrames))
	for k, v := range stats.liveFrames {
		liveFrames[k] = v
	}
	stats.mu.Unlock()

	fmt.Fprintf(os.Stderr, "==== libav allocation stats ====\n")
	fmt.Fprintf(os.Stderr, "packets: created=%d freed=%d live=%d unknown_free=%d\n",
		stats.packetsCreated.Load(), stats.packetsFreed.Load(), len(livePackets), stats.packetsUnknownFree.Load())
	fmt.Fprintf(os.Stderr, "frames:  created=%d freed=%d live=%d unknown_free=%d\n",
		stats.framesCreated.Load(), stats.framesFreed.Load(), len(liveFrames), stats.framesUnknownFree.Load())

	if len(livePackets) > 0 {
		fmt.Fprintf(os.Stderr, "-- live packets by call site --\n")
		printGroups("packet", groupBySite(livePackets, func(id int) string { return fmt.Sprintf("#%d", id) }))
	}
	if len(liveFrames) > 0 {
		fmt.Fprintf(os.Stderr, "-- live frames by call site --\n")
		printGroups("frame", groupBySite(liveFrames, func(p uintptr) string { return fmt.Sprintf("%#x", p) }))
	}
	fmt.Fprintf(os.Stderr, "================================\n")
}

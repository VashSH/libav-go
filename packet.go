package libav

/*
#cgo pkg-config: libavcodec libavformat libavutil
#include <libavcodec/packet.h>
#include <libavcodec/codec_par.h>
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/rational.h>
#include <libavutil/error.h>
#include <string.h>

static inline const char *av_err2str_wrapper(int errnum) { return av_err2str(AVERROR(errnum)); }
*/
import "C"

import (
	"unsafe"
)

type Packet struct {
	inner *C.AVPacket
	data  []byte
	id    int
}

// NewRawPacket allocates an empty AVPacket (no data buffer). Used as a reusable
// output packet for encoders.
func NewRawPacket() (Packet, error) {
	av_pkt := C.av_packet_alloc()
	if av_pkt == nil {
		return Packet{}, ErrOOM
	}
	id := trackPacketAlloc()
	return Packet{av_pkt, nil, id}, nil
}

func NewPacket(data []byte, pts uint64, is_key_frame bool) (Packet, error) {
	av_pkt := C.av_packet_alloc()
	if av_pkt == nil {
		return Packet{}, ErrOOM
	}
	id := trackPacketAlloc()

	av_pkt.data = (*C.uint8_t)(unsafe.Pointer(&data[0]))

	av_pkt.size = C.int(len(data))
	av_pkt.pts = (C.int64_t)(pts)
	av_pkt.dts = av_pkt.pts

	if is_key_frame {
		av_pkt.flags |= C.AV_PKT_FLAG_KEY
	}
	// fmt.Println("new packet with id ", id)
	return Packet{av_pkt, data, id}, nil // hold the data so that the gc can track it
}

func (p *Packet) Set(data []byte, pts int64, is_key_frame bool) {
	p.inner.data = (*C.uint8_t)(unsafe.Pointer(&data[0]))

	p.inner.size = C.int(len(data))
	p.SetPts(pts)
	p.SetDts(pts)

	if is_key_frame {
		p.inner.flags |= C.AV_PKT_FLAG_KEY
	}
}

func (p *Packet) Reset() {
	p.inner.data = nil
	p.inner.size = C.int(0)
	p.SetPts(NoPTSValue)
	p.SetDts(NoPTSValue)
	p.inner.flags = 0
}
func NewPacketAlloc() (Packet, error) {
	av_pkt := C.av_packet_alloc()
	if av_pkt == nil {
		return Packet{}, ErrOOM
	}
	id := trackPacketAlloc()

	return Packet{inner: av_pkt, id: id}, nil
}
func NewPacketRef(avpkt unsafe.Pointer) (Packet, error) {
	pkt := C.av_packet_clone((*C.AVPacket)(avpkt))
	if pkt == nil {
		return Packet{}, ErrOOM
	}
	id := trackPacketAlloc()
	return Packet{inner: pkt, id: id}, nil
}

func NewEofPacket() Packet {
	return WrapAVPacket(nil)
}
func WrapAVPacket(avpkt unsafe.Pointer) Packet {
	return Packet{inner: (*C.AVPacket)(avpkt), data: []byte{}, id: 0}
}

func (p *Packet) Ref() (Packet, error) {
	pkt := C.av_packet_clone((*C.AVPacket)(p.Inner()))
	if pkt == nil {
		return Packet{}, ErrOOM
	}
	id := trackPacketAlloc()
	return Packet{inner: pkt, id: id}, nil
}

func (p *Packet) Unref() {
	// fmt.Println("unrefing packet ", p.id)
	C.av_packet_unref(p.inner)
}

func (p *Packet) Inner() unsafe.Pointer {
	return unsafe.Pointer(p.inner)
}

func (p *Packet) Free() {
	trackPacketFree(p.id)
	C.av_packet_free(&p.inner)
}

func (p *Packet) Data() []byte {
	return unsafe.Slice((*byte)(p.inner.data), p.inner.size)
}

func (p *Packet) TimeBase() Rational {
	return AVRational(unsafe.Pointer(&p.inner.time_base))
}
func (p *Packet) SetTimeBase(timebase Rational) {
	p.inner.time_base = timebase.AVRational()
}

func (p *Packet) Pts() int64 {
	return int64(p.inner.pts)
}

func (p *Packet) SetPts(pts int64) {
	p.inner.pts = (C.int64_t)(pts)
}
func (p *Packet) SetDts(dts int64) {
	p.inner.dts = (C.int64_t)(dts)
}

func (p *Packet) SetStreamIndex(index int) {
	p.inner.stream_index = (C.int)(index)
}

func (p *Packet) StreamIndex() int {
	return (int)(p.inner.stream_index)
}

func (p *Packet) IskeyFrame() bool {
	return (p.inner.flags & C.AV_PKT_FLAG_KEY) != 0
}

// a * b / c
func (p *Packet) Rescale(b, c int64) {
	p.inner.pts = C.av_rescale(p.inner.pts, C.int64_t(b), C.int64_t(c))
	p.inner.dts = p.inner.pts
}

func (p *Packet) RescaleQ(b, c Rational) {
	p.inner.pts = C.av_rescale_q(p.inner.pts, b.AVRational(), c.AVRational())
	p.inner.dts = p.inner.pts
}

func (p *Packet) Duration() int64 {
	return int64(p.inner.duration)
}
func (p *Packet) SetDuration(d int64) {
	p.inner.duration = C.int64_t(d)
}

func (p *Packet) IsValid() bool {
	return !p.IsEAgain() && !p.IsEof()
}

func (p *Packet) IsEAgain() bool {
	return uintptr(unsafe.Pointer(p.inner)) == ^uintptr(0)
}

func (p *Packet) IsEof() bool {
	return p.inner == nil || p.inner.data == nil
}

func (p *Packet) Dts() int64 {
	return int64(p.inner.dts)
}

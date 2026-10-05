package libav

/*
#cgo pkg-config: libavcodec libavformat libavutil
#include <libavcodec/packet.h>
#include <libavcodec/codec_par.h>
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/rational.h>
#include <string.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

var ErrOOM error = errors.New("allocation failure OOM!")
var ErrAvFrameRef error = errors.New("failed to ref avframe!")

type Frame struct {
	inner *C.AVFrame
}

func NewFrame() (Frame, error) {
	frame := C.av_frame_alloc()
	if frame == nil {
		return Frame{}, ErrOOM
	}
	trackFrameAlloc(uintptr(unsafe.Pointer(frame)))
	return Frame{frame}, nil
}

func WrapFrame(f unsafe.Pointer) Frame {
	return Frame{(*C.AVFrame)(f)}
}

func (f Frame) Inner() unsafe.Pointer {
	return unsafe.Pointer(f.inner)
}

func NewEmptyAudioFrame(
	nb_samples int,
	sample_fmt SampleFormat,
	channels int,
) (Frame, error) {
	frame := C.av_frame_alloc()

	frame.nb_samples = C.int(nb_samples)
	frame.format = C.int(sample_fmt)
	C.av_channel_layout_default(&frame.ch_layout, 2)

	frame.pts = 0

	err := C.av_frame_get_buffer(frame, 0)
	if err != 0 {
		panic("failed to get av frame buffer or smth " + string(err))
	}

	C.av_samples_set_silence(
		&frame.data[0],
		0,
		frame.nb_samples,
		frame.ch_layout.nb_channels,
		int32(frame.format),
	)

	if frame == nil {
		return Frame{}, ErrOOM
	}
	trackFrameAlloc(uintptr(unsafe.Pointer(frame)))
	return Frame{frame}, nil

}

func (f Frame) Ref() (Frame, error) {
	if f.inner == nil {
		return f, nil
	}
	ref, err := NewFrame()
	if err != nil {
		return ref, err
	}
	errcode := C.av_frame_ref(ref.inner, f.inner)
	if errcode < 0 {
		ref.Free()
		return ref, fmt.Errorf("%w: (%d) %s", ErrAvFrameRef,
			int(errcode),
			AvErrorString(int(errcode)))
	}
	return ref, nil
}
func (f Frame) Copy() (Frame, error) {
	if f.inner == nil {
		return f, nil
	}

	copied, err := NewFrame()
	if err != nil {
		return copied, err
	}

	// Copy frame properties
	errcode := C.av_frame_copy_props(copied.inner, f.inner)
	if errcode < 0 {
		copied.Free()
		return copied, fmt.Errorf("%w: failed to copy props (%d) %s", ErrAvFrameRef,
			int(errcode),
			AvErrorString(int(errcode)))
	}

	copied.inner.format = f.inner.format
	copied.inner.nb_samples = f.inner.nb_samples
	copied.inner.ch_layout = f.inner.ch_layout
	// Allocate new buffers
	errcode = C.av_frame_get_buffer(copied.inner, 0)
	if errcode < 0 {
		copied.Free()
		return copied, fmt.Errorf("%w: failed to allocate buffers (%d) %s", ErrAvFrameRef,
			int(errcode),
			AvErrorString(int(errcode)))
	}

	// Copy actual buffer data
	errcode = C.av_frame_copy(copied.inner, f.inner)
	if errcode < 0 {
		copied.Free()
		return copied, fmt.Errorf("%w: failed to copy data (%d) %s", ErrAvFrameRef,
			int(errcode),
			AvErrorString(int(errcode)))
	}

	return copied, nil
}

func (f Frame) Unref() {
	C.av_frame_unref(f.inner)
}

func (f *Frame) Free() {
	trackFrameFree(uintptr(unsafe.Pointer(f.inner)))
	C.av_frame_free(&f.inner)
}

func (f *Frame) ExtendedData() (SampleData, int) {
	if f == nil || f.inner == nil {
		return nil, -1
	}

	return toSampleData(f.inner.extended_data, int(f.inner.ch_layout.nb_channels)), int(f.inner.nb_samples)
}

func (f *Frame) SetKeyFrame() {
	f.inner.pict_type = C.AV_PICTURE_TYPE_I
	// f.inner.key_frame = 1
}

func (f Frame) Pts() int64 {
	return int64(f.inner.pts)
}

func (f Frame) SetPts(pts int64) {
	f.inner.pts = C.int64_t(pts)
}

func (f Frame) NbSamples() int64 {
	return int64(f.inner.nb_samples)
}
func (f Frame) Height() int {
	return int(f.inner.height)
}
func (f Frame) Width() int {
	return int(f.inner.width)
}

func (f Frame) IsValid() bool {
	return f.inner != nil
}

func (f Frame) Format() SampleFormat {
	return SampleFormat(f.inner.format)
}
func (f Frame) SampleRate() int {
	return int(f.inner.sample_rate)
}

func (f Frame) NbChannels() int {
	return int(f.inner.ch_layout.nb_channels)
}

func (f Frame) ToAudioInfo() AudioInfo {
	return AudioInfo{
		SampleFmt:  f.Format(),
		SampleRate: f.SampleRate(),
		Channels:   f.NbChannels(),
	}
}

func ReadFrame(ctx AVFormatContext, pkt Packet) int {
	return int(C.av_read_frame(ctx.inner, pkt.inner))
}

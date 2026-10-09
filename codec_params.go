package libav

/*
#include <libavcodec/packet.h>
#include <libavcodec/codec_par.h>
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/rational.h>
#include <libavutil/error.h>
#include <libavutil/mem.h>
#include <string.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type CodecParameters struct {
	inner *C.AVCodecParameters
}

func CodecParametersAlloc() CodecParameters {
	return CodecParameters{C.avcodec_parameters_alloc()}
}
func NewCodecParamters(params unsafe.Pointer) CodecParameters {
	return CodecParameters{(*C.AVCodecParameters)(params)}
}

func (c CodecParameters) IsCodecTypeVideo() bool {
	return c.inner.codec_type == C.AVMEDIA_TYPE_VIDEO
}

func (c CodecParameters) IsCodecTypeAudio() bool {
	return c.inner.codec_type == C.AVMEDIA_TYPE_AUDIO
}

func (c CodecParameters) Id() CodecID {
	return CodecID(c.inner.codec_id)
}

func (c CodecParameters) GetCodecType() MediaType {
	return GetCodecType(c.Id())
}

func (c CodecParameters) IdRaw() int {
	return int(c.inner.codec_id)
}

func (c CodecParameters) SetHeight(height int) {
	c.inner.height = C.int(height)
}
func (c CodecParameters) SetWidth(width int) {
	c.inner.width = C.int(width)
}

func (c CodecParameters) Height() int {
	return int(c.inner.height)
}
func (c CodecParameters) Width() int {
	return int(c.inner.width)
}

func (c CodecParameters) ToAudioInfo() AudioInfo {
	return AudioInfo{
		SampleFmt:  SampleFormat(c.inner.format),
		SampleRate: int(c.inner.sample_rate),
		Channels:   int(c.inner.ch_layout.nb_channels),
	}
}

func (c CodecParameters) SampleFormat() SampleFormat {
	return SampleFormat(c.inner.format)
}

func (c CodecParameters) SampleRate() int {
	return int(c.inner.sample_rate)
}

func (c CodecParameters) BitRate() int {
	return int(c.inner.bit_rate)
}

func (c *CodecParameters) Extradata() []byte {
	p := c.inner
	return unsafe.Slice((*byte)(unsafe.Pointer(p.extradata)), p.extradata_size)
}
func (c *CodecParameters) SetExtradata(data []byte) {
	p := c.inner
	if p.extradata != nil {
		C.av_freep(unsafe.Pointer(&p.extradata))
		p.extradata_size = 0
	}
	if len(data) == 0 {
		return
	}

	// decoders read up to AV_INPUT_BUFFER_PADDING_SIZE bytes past extradata_size
	buffer := C.av_mallocz(C.size_t(len(data)) + C.AV_INPUT_BUFFER_PADDING_SIZE)
	if buffer == nil {
		return
	}
	C.memcpy(buffer, unsafe.Pointer(&data[0]), C.size_t(len(data)))

	p.extradata = (*C.uint8_t)(buffer)
	p.extradata_size = C.int(len(data))
}

func (c *CodecParameters) Free() {
	if c.inner == nil {
		return
	}
	C.avcodec_parameters_free(&c.inner)
}

func (c *CodecParameters) Inner() unsafe.Pointer {
	return unsafe.Pointer(c.inner)
}

func (c CodecParameters) FrameSize() int {
	return int(c.inner.frame_size)
}

func (p *CodecParameters) SetVideo(codec CodecID, width, height, bitrate int) {
	p.inner.codec_type = C.AVMEDIA_TYPE_VIDEO
	p.inner.codec_id = C.enum_AVCodecID(codec)
	p.inner.format = C.AV_PIX_FMT_YUV420P
	p.inner.width = C.int(width)
	p.inner.height = C.int(height)
	p.inner.bit_rate = C.int64_t(bitrate)
}

// change this to take audio info structure or smth idk
func (p *CodecParameters) SetAudio(codec CodecID, sampleRate, channels, frameSize, bitrate int, format AudioFormat) {
	p.inner.codec_type = C.AVMEDIA_TYPE_AUDIO
	p.inner.codec_id = C.enum_AVCodecID(codec)
	p.inner.sample_rate = C.int(sampleRate)
	p.inner.frame_size = C.int(frameSize)
	p.inner.format = C.int(format)

	C.av_channel_layout_default(
		&p.inner.ch_layout,
		C.int(channels),
	)
}

// this is external function which is additional to libav api
// but who cares
// profile-level-id derivation follows ffmpeg libavformat/whip.c
func (p CodecParameters) H264SDPFmtpLine() string {
	codecpar := (*C.AVCodecParameters)(p.inner)

	profileIOP := 0
	if codecpar.profile&C.AV_PROFILE_H264_CONSTRAINED != 0 {
		profileIOP |= 1 << 6
	}
	if codecpar.profile&C.AV_PROFILE_H264_INTRA != 0 {
		profileIOP |= 1 << 4
	}
	profileIDC := int(codecpar.profile) & 0xff
	level := codecpar.level
	return fmt.Sprintf("profile-level-id=%02x%02x%02x", profileIDC, profileIOP, level)
}

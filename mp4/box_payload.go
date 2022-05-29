package mp4

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Reads N bytes from current offset and returns buffer.
func getNBytes(r io.ReadSeeker, n int64) (*bytes.Buffer, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, n))
	_, err := io.CopyN(buffer, r, n)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

// Checks if Nth bit in byte is set.
func checkNthBitSet(bits, n byte) bool {
	res := bits & (1 << n)
	return res != 0
}

// Handler Reference Box's payload structure.
type HdlrPayload struct {
	PreDefined uint32
	Type       BoxType
	Reserved   [3]uint32
	Name       string
}

// Reads Handler Reference Box's payload and returns pointer to it.
func (b *BoxInfo) readHdlr(r io.ReadSeeker) (*HdlrPayload, error) {
	b.toPayload(r)

	buffer, err := getNBytes(r, int64(b.Size)-int64(b.HeaderSize))
	if err != nil {
		return nil, err
	}

	data := buffer.Bytes()

	hdlr := HdlrPayload{
		PreDefined: binary.BigEndian.Uint32(data[4:8]),
		Type:       [4]byte{data[8], data[9], data[10], data[11]},
		Reserved: [3]uint32{
			binary.BigEndian.Uint32(data[12:16]),
			binary.BigEndian.Uint32(data[16:20]),
			binary.BigEndian.Uint32(data[20:24]),
		},
		Name: string(data[24:]),
	}

	return &hdlr, nil
}

// Movie Header Box's payload structure.
type MvhdPayload struct {
	Version          byte
	Flags            [3]byte
	CreationTime     uint64
	ModificationTime uint64
	Timescale        uint32
	Duration         uint64
	Rate             uint32
	Volume           uint16
	Reserved16       uint16
	Reserved32       [2]uint32
	Matrix           [9]uint32
	PreDefined       [6]uint32
	NextTrackId      uint32
}

// Reads Movie Header Box's payload and returns pointer to it.
func (b *BoxInfo) readMvhd(r io.ReadSeeker) (*MvhdPayload, error) {
	b.toPayload(r)

	buffer, err := getNBytes(r, int64(b.Size)-int64(b.HeaderSize))
	if err != nil {
		return nil, err
	}

	data := buffer.Bytes()

	mvhd := MvhdPayload{
		Version: data[0],
		Flags:   [3]byte{data[1], data[2], data[3]},
	}

	var offset uint32

	if mvhd.Version == 1 {
		mvhd.CreationTime = binary.BigEndian.Uint64(data[4:12])
		mvhd.ModificationTime = binary.BigEndian.Uint64(data[12:20])
		mvhd.Timescale = binary.BigEndian.Uint32(data[20:24])
		mvhd.Duration = binary.BigEndian.Uint64(data[24:32])
		offset = 32
	} else {
		mvhd.CreationTime = uint64(binary.BigEndian.Uint32(data[4:8]))
		mvhd.ModificationTime = uint64(binary.BigEndian.Uint32(data[8:12]))
		mvhd.Timescale = binary.BigEndian.Uint32(data[12:16])
		mvhd.Duration = uint64(binary.BigEndian.Uint32(data[16:20]))
		offset = 20
	}

	mvhd.Rate = binary.BigEndian.Uint32(data[offset : offset+4])
	mvhd.Volume = binary.BigEndian.Uint16(data[offset+4 : offset+6])
	mvhd.Reserved16 = binary.BigEndian.Uint16(data[offset+6 : offset+8])
	mvhd.Reserved32[0] = binary.BigEndian.Uint32(data[offset+8 : offset+12])
	mvhd.Reserved32[1] = binary.BigEndian.Uint32(data[offset+12 : offset+16])

	offset += 16
	var i uint32

	for i = 0; i < 9; i++ {
		mvhd.Matrix[i] = binary.BigEndian.Uint32(data[offset+(4*i) : offset+(4*(i+1))])
	}

	offset += 4 * 9

	for i = 0; i < 6; i++ {
		mvhd.PreDefined[i] = binary.BigEndian.Uint32(data[offset+(4*i) : offset+(4*(i+1))])
	}

	offset += 4 * 6

	mvhd.NextTrackId = binary.BigEndian.Uint32(data[offset : offset+4])

	return &mvhd, nil
}

// Sample Description Box's payload structure.
type StsdPayload struct {
	Version byte
	Flags   [3]byte
	Entries uint32
	Codecs  []*BoxInfo
}

// Reads Sample Description Box's payload and returns pointer to it.
func (b *BoxInfo) readStsd(r io.ReadSeeker) (*StsdPayload, error) {
	b.toPayload(r)

	buffer, err := getNBytes(r, 8)
	if err != nil {
		return nil, err
	}

	data := buffer.Bytes()

	stsd := StsdPayload{
		Version: data[0],
		Flags:   [3]byte{data[1], data[2], data[3]},
		Entries: binary.BigEndian.Uint32(data[4:8]),
	}

	var i uint32

	for i = 0; i < stsd.Entries; i++ {
		bi, err := readBoxInfo(r)
		if err != nil {
			return nil, err
		}

		stsd.Codecs = append(stsd.Codecs, bi)
		bi.toNextBox(r)
	}

	return &stsd, nil

}

// Returns all codec config names within Sample Description Box.
func (sp *StsdPayload) getCodecTypes() []string {
	codecs := make([]string, 0)

	for _, c := range sp.Codecs {
		codecs = append(codecs, c.Type.toString())
	}

	return codecs
}

// Track Fragment Header Box's payload structure.
type TfhdPayload struct {
	Version                byte
	Flags                  [3]byte
	TrackId                uint32
	BaseDataOffset         uint64
	SampleDescriptionIndex uint32
	DefaultSampleDuration  uint32
	DefaultSampleSize      uint32
	DefaultSampleFlags     uint32
}

// Reads Track Fragment Header Box's payload and returns pointer to it.
func (b *BoxInfo) readTfhd(r io.ReadSeeker) (*TfhdPayload, error) {
	b.toPayload(r)

	buffer, err := getNBytes(r, int64(b.Size)-int64(b.HeaderSize))
	if err != nil {
		return nil, err
	}

	data := buffer.Bytes()

	tfhd := TfhdPayload{
		Version: data[0],
		Flags:   [3]byte{data[1], data[2], data[3]},
		TrackId: binary.BigEndian.Uint32(data[4:8]),
	}

	var offset uint32
	offset = 8

	if checkNthBitSet(tfhd.Flags[2], 0) {
		tfhd.BaseDataOffset = binary.BigEndian.Uint64(data[offset : offset+8])
		offset += 8
	}

	if checkNthBitSet(tfhd.Flags[2], 1) {
		tfhd.SampleDescriptionIndex = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
	}

	if checkNthBitSet(tfhd.Flags[2], 3) {
		tfhd.DefaultSampleDuration = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
	}

	if checkNthBitSet(tfhd.Flags[2], 4) {
		tfhd.DefaultSampleSize = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
	}

	if checkNthBitSet(tfhd.Flags[2], 5) {
		tfhd.DefaultSampleFlags = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
	}

	return &tfhd, nil
}

// Track Header Box's payload structure.
type TkhdPayload struct {
	Version          byte
	Flags            [3]byte
	CreationTime     uint64
	ModificationTime uint64
	TrackId          uint32
	Reserved32       [3]uint32
	Duration         uint64
	Layer            int16
	AltGroup         int16
	Volume           int16
	Reserved16       uint16
	Matrix           [9]int32
	Width            uint32
	Height           uint32
}

// Reads Track Header Box's payload and returns pointer to it.
func (b *BoxInfo) readTkhd(r io.ReadSeeker) (*TkhdPayload, error) {
	b.toPayload(r)

	buffer, err := getNBytes(r, int64(b.Size)-int64(b.HeaderSize))
	if err != nil {
		return nil, err
	}

	data := buffer.Bytes()

	tkhd := TkhdPayload{
		Version: data[0],
		Flags:   [3]byte{data[1], data[2], data[3]},
	}

	var offset uint16

	if tkhd.Version == 1 {
		tkhd.CreationTime = binary.BigEndian.Uint64(data[4:12])
		tkhd.ModificationTime = binary.BigEndian.Uint64(data[12:20])
		tkhd.TrackId = binary.BigEndian.Uint32(data[20:24])
		tkhd.Reserved32[0] = binary.BigEndian.Uint32(data[24:28])
		tkhd.Duration = binary.BigEndian.Uint64(data[28:36])
		offset = 36
	} else {
		tkhd.CreationTime = uint64(binary.BigEndian.Uint32(data[4:8]))
		tkhd.ModificationTime = uint64(binary.BigEndian.Uint32(data[8:12]))
		tkhd.TrackId = binary.BigEndian.Uint32(data[12:16])
		tkhd.Reserved32[0] = binary.BigEndian.Uint32(data[16:20])
		tkhd.Duration = uint64(binary.BigEndian.Uint32(data[20:24]))
		offset = 24
	}

	tkhd.Reserved32[1] = binary.BigEndian.Uint32(data[offset : offset+4])
	tkhd.Reserved32[2] = binary.BigEndian.Uint32(data[offset+4 : offset+8])
	tkhd.Layer = int16(binary.BigEndian.Uint16(data[offset+8 : offset+10]))
	tkhd.AltGroup = int16(binary.BigEndian.Uint16(data[offset+10 : offset+12]))
	tkhd.Volume = int16(binary.BigEndian.Uint16(data[offset+12 : offset+14]))
	tkhd.Reserved16 = binary.BigEndian.Uint16(data[offset+14 : offset+16])

	offset += 16
	var i uint16
	for i = 0; i < 9; i++ {
		tkhd.Matrix[i] = int32(binary.BigEndian.Uint32(data[offset+(4*i) : offset+(4*(i+1))]))
	}

	offset += 4 * 9

	tkhd.Width = binary.BigEndian.Uint32(data[offset : offset+4])
	tkhd.Width /= 0x10000
	tkhd.Height = binary.BigEndian.Uint32(data[offset+4 : offset+8])
	tkhd.Height /= 0x10000

	return &tkhd, nil
}

// Struct for samples within Track Fragment Run Box.
type Sample struct {
	Duration              uint32
	Size                  uint32
	Flags                 uint32
	CompositionTimeOffset uint32
}

// Track Fragment Run Box's payload structure.
type TrunPayload struct {
	Version          byte
	Flags            [3]byte
	SampleCount      uint32
	Offset           int32
	FirstSampleFlags uint32
	Samples          []*Sample
}

// Reads Track Fragment Run Box's payload and returns pointer to it.
func (b *BoxInfo) readTrun(r io.ReadSeeker) (*TrunPayload, error) {
	b.toPayload(r)

	buffer, err := getNBytes(r, int64(b.Size)-int64(b.HeaderSize))
	if err != nil {
		return nil, err
	}

	data := buffer.Bytes()

	trun := TrunPayload{
		Version:     data[0],
		Flags:       [3]byte{data[1], data[2], data[3]},
		SampleCount: binary.BigEndian.Uint32(data[4:8]),
	}

	var offset uint32
	offset = 8

	if checkNthBitSet(trun.Flags[2], 0) {
		trun.Offset = int32(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
	}

	if checkNthBitSet(trun.Flags[2], 2) {
		trun.FirstSampleFlags = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
	}

	var i uint32

	for i = 0; i < trun.SampleCount; i++ {
		s := Sample{}

		if checkNthBitSet(trun.Flags[1], 0) {
			s.Duration = binary.BigEndian.Uint32(data[offset : offset+4])
			offset += 4
		}

		if checkNthBitSet(trun.Flags[1], 1) {
			s.Size = binary.BigEndian.Uint32(data[offset : offset+4])
			offset += 4
		}

		if checkNthBitSet(trun.Flags[1], 2) {
			s.Flags = binary.BigEndian.Uint32(data[offset : offset+4])
			offset += 4
		}

		if checkNthBitSet(trun.Flags[1], 3) {
			s.CompositionTimeOffset = binary.BigEndian.Uint32(data[offset : offset+4])
			offset += 4
		}

		trun.Samples = append(trun.Samples, &s)
	}

	return &trun, nil
}

// Returns summarized duration of all samples
func (tp *TrunPayload) getDurationFromSamples() uint64 {
	dur := uint64(0)
	for _, s := range tp.Samples {
		dur += uint64(s.Duration)
	}
	return dur
}

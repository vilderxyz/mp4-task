package mp4

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	HeaderSize         = 8
	ExtendedHeaderSize = 16
)

type BoxInfo struct {
	Type        BoxType
	Size        uint64
	Offset      uint64
	HeaderSize  uint32
	HasChildren bool
}

func readBoxInfo(r io.ReadSeeker) (*BoxInfo, error) {
	offset, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	bi := &BoxInfo{
		Offset:     uint64(offset),
		HeaderSize: 8,
	}

	buffer, err := getNBytes(r, HeaderSize)
	if err != nil {
		return nil, err
	}

	bytes := buffer.Bytes()
	bi.Size = uint64(binary.BigEndian.Uint32(bytes[0:4]))
	bi.Type = BoxType{bytes[4], bytes[5], bytes[6], bytes[7]}

	bi.HasChildren = SupportedBoxTypes[bi.Type.toString()]

	if bi.Size == 0 {
		eof, err := r.Seek(0, io.SeekEnd)

		_, err = r.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, err
		}

		bi.Size = uint64(eof) - bi.Offset

	} else if bi.Size == 1 {
		buffer.Reset()

		_, err := io.CopyN(buffer, r, 8)
		if err != nil {
			return nil, err
		}

		bi.HeaderSize = ExtendedHeaderSize
		bi.Size = binary.BigEndian.Uint64(buffer.Bytes())
	}

	return bi, nil
}

func (b *BoxInfo) readStructure(r io.ReadSeeker, i *Mp4Struct) (uint64, error) {
	err := b.readPayload(r, i)
	if err != nil {
		return b.Size, err
	}

	if !b.HasChildren {
		b.toNextBox(r)
		return b.Size, nil
	}

	currentOffset := uint64(b.HeaderSize)
	var offset uint64

	for currentOffset < b.Size {
		bi, err := readBoxInfo(r)
		if err != nil {
			return 0, err
		}

		if _, ok := SupportedBoxTypes[bi.Type.toString()]; ok {
			// bi.Print()

			offset, err = bi.readStructure(r, i)
			if err != nil {
				return offset, err
			}
		} else {
			bi.toNextBox(r)
			offset = bi.Size
		}

		currentOffset += offset
	}

	return currentOffset, nil
}

func (b *BoxInfo) readPayload(r io.ReadSeeker, i *Mp4Struct) error {
	switch b.Type.toString() {
	case "traf":
		i.MovieFragments = append(i.MovieFragments, &MovieFragment{})
		return nil
	case "trak":
		i.Tracks = append(i.Tracks, &Track{})
		return nil
	case "hdlr":
		hr, err := b.readHdlr(r)
		if err != nil {
			return err
		}

		i.Tracks[len(i.Tracks)-1].MediaHandler = hr
		return nil
	case "tkhd":
		th, err := b.readTkhd(r)
		if err != nil {
			return err
		}

		i.Tracks[len(i.Tracks)-1].TrackHeader = th
		return nil
	case "stsd":
		sd, err := b.readStsd(r)
		if err != nil {
			return err
		}

		i.Tracks[len(i.Tracks)-1].SampleDescription = sd
		return nil
	case "mvhd":
		mh, err := b.readMvhd(r)
		if err != nil {
			return err
		}

		i.MovieHeader = mh
		return nil
	case "tfhd":
		tfh, err := b.readTfhd(r)
		if err != nil {
			return err
		}

		i.MovieFragments[len(i.MovieFragments)-1].TrackFragmentHeader = tfh
		return nil
	case "trun":
		tfr, err := b.readTrun(r)
		if err != nil {
			return err
		}

		i.MovieFragments[len(i.MovieFragments)-1].TrackFragmentRun = tfr
		return nil
	default:
		return nil
	}
}

func (bi *BoxInfo) Print() {
	fmt.Printf("Type: %s, offset: %d, size: %d\n", bi.Type.toString(), bi.Offset, bi.Size)
}

func (bi *BoxInfo) toNextBox(s io.Seeker) (int64, error) {
	return s.Seek(int64(bi.Offset+bi.Size), io.SeekStart)
}

func (bi *BoxInfo) toPayload(s io.Seeker) (int64, error) {
	return s.Seek(int64(bi.Offset+uint64(bi.HeaderSize)), io.SeekStart)
}

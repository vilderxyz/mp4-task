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

// Main structure of the Box in this package. It's built from reading Box's header.
type BoxInfo struct {
	// Type of the Box. Built from subtype [4]byte.
	Type BoxType
	// Size of the Box. Summary of both header and payload bytes.
	Size uint64
	// Distance between the beginning of the Box and MP4 file.
	Offset uint64
	// Size of the Box's header.
	HeaderSize uint32
	// Is set to true if Box has other Boxes within its payload
	HasChildren bool
}

// Reads Box's header from the current offset in the file and creates BoxInfo object.
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

	// If Box's size is equal to 0 then it is the last one in file.
	if bi.Size == 0 {
		eof, err := r.Seek(0, io.SeekEnd)

		_, err = r.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, err
		}

		bi.Size = uint64(eof) - bi.Offset

		// If Box's size is equeal to 1 then its size extended and is stored in next 8 bytes.
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

// Reads Box's children or payload if specified. Otherwise returns only a Box's size and skips to the next one.
func (b *BoxInfo) readStructure(r io.ReadSeeker, i *ReadPayloads) (uint64, error) {
	err := b.readPayload(r, i)
	if err != nil {
		return b.Size, err
	}

	if !b.HasChildren {
		b.toNextBox(r)
		return b.Size, nil
	}

	currentOffset := uint64(b.HeaderSize)

	// Loops through all children and stops when their summarize size is equal to this Box's size.
	for currentOffset < b.Size {
		bi, err := readBoxInfo(r)
		if err != nil {
			return 0, err
		}

		var size uint64

		if _, ok := SupportedBoxTypes[bi.Type.toString()]; ok {
			// bi.Print()

			size, err = bi.readStructure(r, i)
			if err != nil {
				return size, err
			}
		} else {
			bi.toNextBox(r)
			size = bi.Size
		}

		currentOffset += size
	}

	return currentOffset, nil
}

// Commits different actions on Box's payload according to its type.
func (b *BoxInfo) readPayload(r io.ReadSeeker, i *ReadPayloads) error {
	switch b.Type.toString() {
	// Appends new MovieFragment object to the slice. Sub-Boxes can them refer to it and pass references to their payloads.
	case "traf":
		i.MovieFragments = append(i.MovieFragments, &MovieFragment{})
		return nil
	// Appends new Track object to the slice. Sub-Boxes can them refer to it and pass references to their payloads.
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

// Prints BoxInfo variables.
func (bi *BoxInfo) Print() {
	fmt.Printf("Type: %s, offset: %d, size: %d, header's size: %d, children: %v\n", bi.Type.toString(), bi.Offset, bi.Size, bi.HeaderSize, bi.HasChildren)
}

// Skips Box's content and sets offset to the next Box.
func (bi *BoxInfo) toNextBox(s io.Seeker) (int64, error) {
	return s.Seek(int64(bi.Offset+bi.Size), io.SeekStart)
}

// Sets offset to Box's payload.
func (bi *BoxInfo) toPayload(s io.Seeker) (int64, error) {
	return s.Seek(int64(bi.Offset+uint64(bi.HeaderSize)), io.SeekStart)
}

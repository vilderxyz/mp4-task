package mp4

import (
	"io"
)

// Represents single Track within Movie Box.
type Track struct {
	TrackHeader       *TkhdPayload
	MediaHandler      *HdlrPayload
	SampleDescription *StsdPayload
}

// Represents a single MovieFragment that extends the presentation in time.
type MovieFragment struct {
	TrackFragmentHeader *TfhdPayload
	TrackFragmentRun    *TrunPayload
}

// Represents all read playloads and provide access to them at every level.
type ReadPayloads struct {
	MovieHeader    *MvhdPayload
	Tracks         []*Track
	MovieFragments []*MovieFragment
}

// Reads MP4 file boxes, stores all read payloads in struct and returns pointer to it.
func getPayloads(r io.ReadSeeker, bi *BoxInfo) (*ReadPayloads, error) {
	rs := ReadPayloads{}

	_, err := bi.readStructure(r, &rs)
	if err != nil {
		return &rs, err
	}

	return &rs, nil
}

// Returns all media types within file.
func (s *ReadPayloads) getMediaTypes() []string {
	types := []string{}

	for _, t := range s.Tracks {
		types = append(types, string(t.MediaHandler.Type.toString()))
	}

	return types
}

// Returns all supported codecs.
func (s *ReadPayloads) getSupportedCodecs() []string {
	codecs := []string{}

	for _, t := range s.Tracks {
		codecs = append(codecs, t.SampleDescription.getCodecTypes()...)
	}

	return codecs
}

// Returns video resolution.
func (s *ReadPayloads) getVideoResolution() (w uint32, h uint32) {
	for _, t := range s.Tracks {
		if t.MediaHandler.Type.toString() == "vide" {
			w = t.TrackHeader.Width
			h = t.TrackHeader.Height
			return
		}
	}

	return 0, 0
}

// Returns presentation duration.
func (s *ReadPayloads) getDuration() float64 {
	d := uint64(0)

	d += s.MovieHeader.Duration

	for _, f := range s.MovieFragments {
		if checkNthBitSet(f.TrackFragmentHeader.Flags[2], 3) {
			d += uint64(f.TrackFragmentHeader.DefaultSampleDuration) * uint64(f.TrackFragmentRun.SampleCount)
		} else {
			d += f.TrackFragmentRun.getDurationFromSamples()
		}
	}

	return float64(d) / float64(s.MovieHeader.Timescale)
}

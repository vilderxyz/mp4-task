package mp4

import (
	"io"
)

type Track struct {
	TrackHeader       *TkhdPayload
	MediaHandler      *HdlrPayload
	SampleDescription *StsdPayload
}

type MovieFragment struct {
	TrackFragmentHeader *TfhdPayload
	TrackFragmentRun    *TrunPayload
}

type Mp4Struct struct {
	MovieHeader    *MvhdPayload
	Tracks         []*Track
	MovieFragments []*MovieFragment
}

func getStruct(r io.ReadSeeker, bi *BoxInfo) (*Mp4Struct, error) {
	rs := Mp4Struct{}

	_, err := bi.readStructure(r, &rs)
	if err != nil {
		return &rs, err
	}

	return &rs, nil
}

func (s *Mp4Struct) getMediaTypes() []string {
	types := []string{}

	for _, t := range s.Tracks {
		types = append(types, string(t.MediaHandler.Type.toString()))
	}

	return types
}

func (s *Mp4Struct) getSupportedCodecs() []string {
	codecs := []string{}

	for _, t := range s.Tracks {
		codecs = append(codecs, t.SampleDescription.getCodecTypes()...)
	}

	return codecs
}

func (s *Mp4Struct) getVideoResolution() (w uint32, h uint32) {
	for _, t := range s.Tracks {
		if t.MediaHandler.Type.toString() == "vide" {
			w = t.TrackHeader.Width
			h = t.TrackHeader.Height
			return
		}
	}

	return 0, 0
}

func (s *Mp4Struct) getDuration() float64 {
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

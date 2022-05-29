package mp4

import "io"

// Represents necessary information about MP4 file.
type Info struct {
	MediaTypes  []string
	Codecs      []string
	VideoHeight uint32
	VideoWidth  uint32
	Duration    float64
}

// Returns all required information about MP4 file.
func GetInfo(r io.ReadSeeker, bi *BoxInfo) (*Info, error) {
	mp4s, err := getPayloads(r, bi)
	if err != nil {
		return nil, err
	}

	w, h := mp4s.getVideoResolution()

	info := Info{
		MediaTypes:  mp4s.getMediaTypes(),
		Codecs:      mp4s.getSupportedCodecs(),
		VideoWidth:  w,
		VideoHeight: h,
		Duration:    mp4s.getDuration(),
	}

	return &info, nil
}

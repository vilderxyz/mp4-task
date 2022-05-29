package mp4

import "io"

type Mp4Info struct {
	MediaTypes  []string
	Codecs      []string
	VideoHeight uint32
	VideoWidth  uint32
	Duration    float64
}

func GetInfo(r io.ReadSeeker, bi *BoxInfo) (*Mp4Info, error) {
	mp4s, err := getStruct(r, bi)
	if err != nil {
		return nil, err
	}

	w, h := mp4s.getVideoResolution()

	info := Mp4Info{
		MediaTypes:  mp4s.getMediaTypes(),
		Codecs:      mp4s.getSupportedCodecs(),
		VideoWidth:  w,
		VideoHeight: h,
		Duration:    mp4s.getDuration(),
	}

	return &info, nil
}

package mp4

// All supported Boxes and if they have other boxes within.
var SupportedBoxTypes = map[string]bool{
	"moov": true,
	"mvhd": false,
	"trak": true,
	"tkhd": false,
	"mdia": true,
	"hdlr": false,
	"minf": true,
	"stbl": true,
	"stsd": false,
	"mvex": true,
	"moof": true,
	"traf": true,
	"tfhd": false,
	"trun": false,
}

type BoxType [4]byte

// Returns Box's type as string.
func (bt BoxType) toString() string {
	return string([]byte{bt[0], bt[1], bt[2], bt[3]})
}

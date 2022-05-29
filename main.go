package main

import (
	"fmt"
	"log"
	"os"

	"github.com/vilderxyz/mp4-task/mp4"
)

func main() {
	// Enter valid path to mp4 file
	// I'm sorry about lack of proper UI.
	file, err := os.Open("./samples/s3.mp4")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	root := mp4.BoxInfo{
		Type:        mp4.BoxType{0, 0, 0, 0},
		Size:        uint64(fi.Size()),
		HeaderSize:  0,
		HasChildren: true,
	}

	info, err := mp4.GetInfo(file, &root)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Media types: %v\n", info.MediaTypes)
	fmt.Printf("Supported codecs: %v\n", info.Codecs)
	fmt.Printf("Video resolution: %dx%d\n", info.VideoWidth, info.VideoHeight)
	fmt.Printf("Duration: %.2fs\n", info.Duration)
}

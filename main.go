package main

import (
	"bitDownloader/parser"
	"log"
	"os"
)

func main() {
	inpath, err := os.Open("testdata/test.torrent")
	if err != nil {
		panic(err)
	}

	tf, err := parser.Open(inpath)
	tof, err := tf.ToTorrentFile()

	if err != nil {
		log.Fatal(err)
	}

	err = tof.DownloadToFile("result/test.mp4")
	if err != nil {
		log.Fatal(err)
	}
}

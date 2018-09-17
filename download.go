package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"math/bits"
	"net/http"
	"os"
)

func downloadFile(src, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer out.Close()

	log.Printf("starting download: %s\n", src)

	done := make(chan int64)

	go printDownloadPercent(done, dst)

	res, err := http.Get(src)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	n, err := io.Copy(out, res.Body)
	if err != nil {
		return err
	}

	done <- n

	log.Printf("finished download: %s\n", dst)

	return nil
}

func printDownloadPercent(done chan int64, path string) {
	var stop = false

	for {
		select {
		case <-done:
			stop = true
		default:
			file, err := os.Open(path)
			if err != nil {
				log.Fatal(err)
			}

			fi, err := file.Stat()
			if err != nil {
				log.Fatal(err)
			}

			size := fi.Size()

			// Return and rewrite the download status in a human readable form.
			fmt.Printf("\rDownloaded %s", humanBytes(uint64(size)))
		}

		if stop {
			// Add a newline once the download finishes.
			fmt.Printf("\n")
			break
		}
	}
}

// Convert bytes to a human readable format.
func humanBytes(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d bytes", bytes)
	}

	base := uint(bits.Len64(bytes) / 10)
	val := float64(bytes) / math.Pow(2, float64(base*10))

	return fmt.Sprintf("%.1f %ciB", val, " KMGTPE"[base])
}

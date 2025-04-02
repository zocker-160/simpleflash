package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	gzip "github.com/klauspost/pgzip"
	"github.com/schollz/progressbar/v3"
	"github.com/ulikunitz/xz"
)

func imageToImage() error {
	file, err := getImageFile()
	if err != nil {
		return err
	}

	fileH, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fileH.Close()

	fileInfo, err := fileH.Stat()
	if err != nil {
		return err
	}
	fileSize := fileInfo.Size()

	var compressed bool
	var targetFile string
	var source io.Reader

	bar := progressbar.DefaultBytes(fileSize)

	if strings.HasSuffix(file, XZ_SUFFIX) {
		fmt.Println("detected xz compression")
		targetFile = strings.TrimSuffix(file, XZ_SUFFIX)
		compressed = true
 
		r, err := xz.NewReader(io.TeeReader(fileH, bar))
		if err != nil {
			return err
		}

		source = r
	} else if strings.HasSuffix(file, GZ_SUFFIG) {
		fmt.Println("detected gzip compression")
		targetFile = strings.TrimSuffix(file, GZ_SUFFIG)
		compressed = true

		r, err := gzip.NewReader(io.TeeReader(fileH, bar))
		if err != nil {
			return err
		}
		defer r.Close()

		source = r
	} else {
		compressed = false
		targetFile = addSuffix(file, GZ_SUFFIG)
		source = io.TeeReader(fileH, bar)
	}

	target, err := os.OpenFile(targetFile, os.O_WRONLY | os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer target.Close()

	if compressed {
		bar.Describe("Decompressing image")

		_, err := io.Copy(target, source)
		if err != nil {
			return err
		}
	} else {
		bar.Describe("Compressing image")

		tc := gzip.NewWriter(target)
		defer tc.Close()

		_, err := io.Copy(tc, source)
		if err != nil {
			return err
		}
		
	}

	return nil
}

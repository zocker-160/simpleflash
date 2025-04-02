package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/schollz/progressbar/v3"

	"github.com/ulikunitz/xz"
	gzip "github.com/klauspost/pgzip"
)


func flashImageToDisk() error {
	file, err := getImageFile()
	if err != nil {
		return err
	}

	_, target, err := getDisk(false, true)
	if err != nil {
		return err
	}

	question := fmt.Sprintf(
		"Are you sure you want to flash \"%s\" to \"%s\"? ALL DATA ON \"%s\" WILL BE LOST!", 
		filepath.Base(file), target, target,
	)
	confirm := huh.NewConfirm().Title(question).Affirmative("Yes").Negative("NO")
	if err := confirm.Run(); err != nil {
		return err
	}

	if confirm.GetValue() == true {
		if err := flashImage(file, target, DefaultBuffsize); err != nil {
			return err
		}
	} else {
		main()
	}

	return nil
}

func flashImage(imagePath, target string, buffsize Buffsize) error {
	imgFile, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer imgFile.Close()

	imgInfo, err := imgFile.Stat()
	if err != nil {
		return err
	}

	targetFile, err := os.OpenFile(target, os.O_WRONLY | os.O_CREATE | os.O_SYNC, 0644)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	var source io.Reader
	bar := progressbar.DefaultBytes(imgInfo.Size(), "Flashing")

	if strings.HasSuffix(imagePath, XZ_SUFFIX) {
		fmt.Println("detected xz compression")
		r, err := xz.NewReader(io.TeeReader(imgFile, bar))
		if err != nil {
			return err
		}
		source = r
	} else if strings.HasSuffix(imagePath, GZ_SUFFIG) {
		fmt.Println("detected gzip compression")
		r, err := gzip.NewReader(io.TeeReader(imgFile, bar))
		if err != nil {
			return err
		}
		defer r.Close()

		source = r
	} else {
		source = io.TeeReader(imgFile, bar)
	}

	written, err := copy(targetFile, source, buffsize)
	if err != nil {
		return err
	}

	if err := targetFile.Sync(); err != nil {
		return err
	}

	if written != imgInfo.Size() {
		return fmt.Errorf("written size does not equal expected size")
	}

	return nil
}
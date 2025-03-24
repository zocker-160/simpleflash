package main

import (
	"context"
	"fmt"
	"io"
	"os"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/schollz/progressbar/v3"
)


func createImageFromDisk() error {
	var showInternal bool
	err := huh.NewConfirm().
		Title("Do you want to list internal drives?").
		Affirmative("Yes").
		Negative("NO").
		Value(&showInternal).
		Run()
	if err != nil {
		return err
	}

	disk, source, err := getDisk(true, !showInternal)
	if err != nil {
		return err
	}

	target, err := getImageSaveLoc(disk.Model)
	if err != nil {
		return err
	}

	question := fmt.Sprintf("Are you sure you want to create \"%s\" from \"%s\"?", target, source)
	confirm := huh.NewConfirm().Title(question).Affirmative("Yes").Negative("NO")
	if err := confirm.Run(); err != nil {
		return err
	}

	if confirm.GetValue() == true {
		if err := createImage(source, target, int64(disk.SizeBytes)); err != nil {
			return err
		}
	} else {
		main()
	}

	return nil
}

func createImage(source, imagePath string, rawSize int64) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	var compress bool
	err = huh.NewConfirm().
		Title("Would you like to enable compression (gzip)?").
		Affirmative("Yes").
		Negative("No").
		Value(&compress).
		Run()

	if err != nil {
		return err
	}

	if compress {
		imagePath = imagePath + ".gz"
	}

	targetFile, err := os.OpenFile(imagePath, os.O_WRONLY | os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	var written int64

	if compress {
		bar := progressbar.DefaultBytes(rawSize, "Creating compressed image")
		targetComp := gzip.NewWriter(targetFile)

		written, err = io.Copy(io.MultiWriter(targetComp, bar), srcFile)
		if err != nil {
			return err
		}
		targetComp.Close()
	} else {
		bar := progressbar.DefaultBytes(rawSize, "Creating image")

		written, err = io.Copy(io.MultiWriter(targetFile, bar), srcFile)
		if err != nil {
			return err
		}
	}

	err = spinner.New().
		Title("Waiting for file sync to finish").
		ActionWithErr(func(ctx context.Context) error {
			return targetFile.Sync()
		}).
		Run()

	if err != nil {
		return err
	}

	if written != rawSize {
		return fmt.Errorf("written size does not equal expected size")
	}

	return nil
}

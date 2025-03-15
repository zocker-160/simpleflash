package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/rymdport/portal/filechooser"
	"github.com/schollz/progressbar/v3"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"

	"github.com/inhies/go-bytesize"

	_ "github.com/ulikunitz/xz"
	gzip "github.com/klauspost/pgzip"
)


const BUFFSIZE = 1024*1024*1 // 1MB


func main() {
	checkRoot()

	modeSelect := huh.NewSelect[int]().Title("What do you want to do?").Options(
			huh.NewOption("Flash / Restore image file to USB drive", 0),
			huh.NewOption("Backup USB drive to image file", 1),
			huh.NewOption("Exit", 2),
		)

	err := modeSelect.Run()
	if err != nil {
		panic(err)
	}

	switch modeSelect.GetValue() {
	case 0:
		err = flashImageToDisk()
	case 1:
		err = createImageFromDisk()
	case 2:
		os.Exit(0)
	}

	if err != nil {
		fmt.Println("Error:", err)
		main()
	}
}

func checkRoot() {
	if os.Geteuid() != 0 {
		fmt.Println("This program must be run as root.")
		os.Exit(1)
	}
}


func flashImageToDisk() error {
	file, err := getImageFile()
	if err != nil {
		return err
	}

	_, target, err := getDisk(false)
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
		if err := flashImage(file, target, BUFFSIZE); err != nil {
			return err
		}
	} else {
		main()
	}

	return nil
}

func createImageFromDisk() error {
	disk, source, err := getDisk(true)
	if err != nil {
		return err
	}

	target, err := getImageSaveLoc(disk.Model)
	if err != nil {
		return err
	}

	question := fmt.Sprintf("Are you sure you want to backup \"%s\" to \"%s\"?", source, target)
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

func getImageSaveLoc(deviceName string) (string, error) {
	if len(os.Args) > 1 {
		return os.Args[1], nil
	}

	options := filechooser.SaveFileOptions{
		CurrentName: deviceName,
		Filters: []*filechooser.Filter{
			{
				Name: "Image file (.img)",
				Rules: []filechooser.Rule{{Type: filechooser.GlobPattern, Pattern: "*.img"}},
			},
		},
	}
	files, err := filechooser.SaveFile("", "Save disk as image", &options)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files selected")
	}

	file, _ := strings.CutPrefix(files[0], "file://")
	return file, nil
}

func getImageFile() (string, error) {
	if len(os.Args) > 1 {
		return os.Args[1], nil
	}

	options := filechooser.OpenFileOptions{
		Multiple: false,
		CurrentFolder: filepath.Dir(os.Args[0]),
		Filters: []*filechooser.Filter{
			{
				Name: "Image files (.iso/.img)",
				Rules: []filechooser.Rule{
					{Type: filechooser.GlobPattern, Pattern: "*.img"},
					{Type: filechooser.GlobPattern, Pattern: "*.iso"},
				},
			},
			{
				Name: "Compressed image files (.gz/.img.gz)",
				Rules: []filechooser.Rule{
					{Type: filechooser.GlobPattern, Pattern: "*.gz"},
					{Type: filechooser.GlobPattern, Pattern: "*.img.gz"},
				},
			},
			{
				Name: "All files",
				Rules: []filechooser.Rule{{Type: filechooser.GlobPattern, Pattern: "*.*"}},
			},
		},
	}
	files, err := filechooser.OpenFile("", "Select image file", &options)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files selected")
	}

	
	file := strings.TrimPrefix(files[0], "file://")
	return file, nil
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

		/*
		targetComp, err := xz.NewWriter(targetFile)
		if err != nil {
			return err
		}
		*/

		written, err = io.Copy(io.MultiWriter(targetComp, bar), srcFile)
		if err != nil {
			return err
		}
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

func flashImage(imagePath, target string, buffsize int) error {
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

	bar := progressbar.DefaultBytes(imgInfo.Size(), "Flashing")
	written, err := copy(io.MultiWriter(targetFile, bar), imgFile, buffsize)

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

func getDisk(source bool) (*ghw.Disk, string, error) {
	block, err := ghw.Block()
	if err != nil {
		return nil, "", err
	}

	var selection *ghw.Disk
	options := make([]huh.Option[*ghw.Disk], 0)

	for _, disk := range block.Disks {
		if !disk.IsRemovable || disk.DriveType == ghw.DriveTypeODD {
			continue
		}

		size := bytesize.New(float64(disk.SizeBytes))
		name := fmt.Sprintf("%s %s %s (/dev/%s)", disk.Vendor, disk.Model, size, disk.Name)

		option := huh.NewOption(name, disk)
		options = append(options, option)
	}

	if len(options) == 0 {
		return nil, "", fmt.Errorf("no suitable disk found")
	}

	var role string
	if source {
		role = "source"
	} else {
		role = "target"
	}

	err = huh.NewSelect[*ghw.Disk]().
		Title(fmt.Sprintf("Select %s disk:", role)).
		Options(options...).
		Value(&selection).
		Run()

	if err != nil {
		return nil, "", err
	}

	return selection, fmt.Sprintf("/dev/%s", selection.Name), nil
}


func copy(dst io.Writer, src io.Reader, buffsize int) (int64, error) {
	buffer := make([]byte, buffsize)
	written := int64(0)

	for {
		n, err := src.Read(buffer)
		if errors.Is(err, io.EOF) || n == 0 {
			break
		}
		if err != nil {
			return written, err
		}

		wn, err := dst.Write(buffer[:n])
		if err != nil {
			return written, err
		}

		written += int64(wn)
	}

	return written, nil
}

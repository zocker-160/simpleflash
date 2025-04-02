package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/inhies/go-bytesize"
	"github.com/jaypipes/ghw"
	"github.com/rymdport/portal/filechooser"
)

const IMG_SUFFIX = ".img"
const XZ_SUFFIX = ".xz"
const GZ_SUFFIG = ".gz"


func getImageSaveLoc(deviceName string) (string, error) {
	if len(os.Args) > 1 {
		fname := addSuffix(os.Args[1], IMG_SUFFIX)
		return fname, nil
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
		fmt.Println("Failed to open file chooser portal", err)
		
		var fname string
		err := huh.NewInput().Title("Output filename:").Value(&fname).Run()
		if err != nil {
			return "", err
		}
		if len(fname) == 0 {
			return "", fmt.Errorf("no name specified")
		}

		fname = addSuffix(fname, IMG_SUFFIX)
		return fname, nil
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
				Name: "GZIP compressed image files (.gz/.img.gz)",
				Rules: []filechooser.Rule{
					{Type: filechooser.GlobPattern, Pattern: "*.gz"},
					{Type: filechooser.GlobPattern, Pattern: "*.img.gz"},
				},
			},
			{
				Name: "XZ compressed image files (.xz/.img.xz)",
				Rules: []filechooser.Rule{
					{Type: filechooser.GlobPattern, Pattern: "*.xz"},
					{Type: filechooser.GlobPattern, Pattern: "*.img.xz"},
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


func getDisk(source, removableOnly bool) (*ghw.Disk, string, error) {
	block, err := ghw.Block()
	if err != nil {
		return nil, "", err
	}

	var selection *ghw.Disk
	options := make([]huh.Option[*ghw.Disk], 0)

	for _, disk := range block.Disks {
		if disk.DriveType == ghw.DriveTypeODD {
			continue
		}
		if removableOnly && !disk.IsRemovable {
			continue
		}
		if strings.HasPrefix(disk.Name, "ram") {
			continue
		}

		size := bytesize.New(float64(disk.SizeBytes))
		name := fmt.Sprintf("%s %s %s", disk.Name, size, strings.ReplaceAll(disk.Model, "_", " "))
		if disk.IsRemovable {
			name += " (removable)"
		}

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

func checkOS() {
	if runtime.GOOS != "linux" {
		fmt.Println("Unsupported OS")
		os.Exit(1)
	}
}

func checkRoot() bool {
	return os.Geteuid() == 0
}

func addSuffix(s, suffix string) string {
	if !strings.HasSuffix(s, suffix) {
		s += suffix
	}
	return s
}

func copy(dst io.Writer, src io.Reader, buffsize Buffsize) (int64, error) {
	buffer := make([]byte, int(buffsize))
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

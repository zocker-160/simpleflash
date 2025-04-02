package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)


type Buffsize int
const (
	Byte Buffsize = 1
	Kilobyte Buffsize = Byte * 1024
	Megabyte Buffsize = Kilobyte * 1024
)

const DefaultBuffsize = 1 * Megabyte


func main() {
	checkOS()
	isRoot := checkRoot()

	modeSelect := huh.NewSelect[int]().
		Title("Select action").
		Options(
			huh.NewOption("Flash image file to drive", 0),
			huh.NewOption("Create image file from drive", 1),
			huh.NewOption("Image to image conversion", 3),
			huh.NewOption("Transer image file over SSH (WiP)", 2),
			huh.NewOption("Exit", 99),
		)

	err := modeSelect.Run()
	if err != nil {
		return
	}

	switch modeSelect.GetValue() {
	case 0:
		err = requireRoot(isRoot, flashImageToDisk)
	case 1:
		err = requireRoot(isRoot, createImageFromDisk)
	case 2:
		err = requireRoot(isRoot, createImageToSSH)
	case 3:
		err = imageToImage()

	case 99:
		return
	}

	if err != nil {
		fmt.Println(err)
		main()
	}
}

func requireRoot(isRoot bool, action func() error) error {
	if isRoot {
		return action()
	} else {
		fmt.Println("This action must be run as root")
		os.Exit(1)
		return nil
	}
}

package main

import (
	"fmt"

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
	checkRoot()

	modeSelect := huh.NewSelect[int]().
		Title("Select action").
		Options(
			huh.NewOption("Flash image file to drive", 0),
			huh.NewOption("Create image file from drive", 1),
			huh.NewOption("Transer image file over SSH (WiP)", 2),
			huh.NewOption("Exit", 99),
		)

	err := modeSelect.Run()
	if err != nil {
		return
	}

	switch modeSelect.GetValue() {
	case 0:
		err = flashImageToDisk()
	case 1:
		err = createImageFromDisk()
	case 2:
		err = createImageToSSH()

	case 99:
		return
	}

	if err != nil {
		fmt.Println(err)
		main()
	}
}

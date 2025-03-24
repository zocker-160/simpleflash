package main

import (
	"fmt"
	"io"
	"os"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"github.com/gliderlabs/ssh"
	"github.com/schollz/progressbar/v3"
)


func createImageToSSH() error {
	disk, source, err := getDisk(true, false)
	if err != nil {
		return err
	}

	fmt.Println("waiting for incoming connection on port 2222...")
	fmt.Printf(
		"NOTE: ssh -o \"StrictHostKeyChecking=no\" -q <serverIP> -p 2222 > %s.img.gz",
		disk.Model,
	)

	return ssh.ListenAndServe(
		":2222", 
		handleIncomingSSH(source, int64(disk.SizeBytes)), 
		ssh.NoPty(),
	)
}

func handleIncomingSSH(source string, rawSize int64) ssh.Handler {
	return func(session ssh.Session) {
		defer fmt.Println("connection closed (Ctrl + C to exit)")
		defer session.Close()
	
		srcFile, err := os.Open(source)
		if err != nil {
			return
		}
		defer srcFile.Close()
	
		bar := progressbar.DefaultBytes(rawSize, "Transferring over SSH")
		targetComp := gzip.NewWriter(session)

		written, err := io.Copy(io.MultiWriter(targetComp, bar), srcFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		targetComp.Close()
	
		fmt.Println(written, "bytes sent")
	}
}

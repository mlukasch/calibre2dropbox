package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	fmt.Println()
	fmt.Println("Kopieren aller konvertierten Pdfs der x Minuten von Calibre nach Dropbox")
	fmt.Println()
	fmt.Println("Bitte Zielordner in Dropbox angeben:")
	reader := bufio.NewReader(os.Stdin)
	input, _, _ := reader.ReadLine()
	destFolderName := strings.TrimSpace(string(input))
	fmt.Println()
	fmt.Println("Bitte Delta in min angeben für Pdfs, die kopiert werden sollen: 60 (default)")
	input, _, _ = reader.ReadLine()
	deltaInMins := 60
	if delta := strings.TrimSpace(string(input)); len(delta) != 0 {
		deltaInt64, err := strconv.ParseInt(delta, 0, 0)
		if err != nil {
			panic(err)
		}
		deltaInMins = int(deltaInt64)
	}
	fmt.Println()

	pdfFiles := make(chan string)
	go pdfFilesProducer(pdfFiles, int(deltaInMins))

	var wg sync.WaitGroup
	wg.Add(4)
	go consumePdfFiles(pdfFiles, destFolderName, &wg)
	go consumePdfFiles(pdfFiles, destFolderName, &wg)
	go consumePdfFiles(pdfFiles, destFolderName, &wg)
	go consumePdfFiles(pdfFiles, destFolderName, &wg)
	wg.Wait()
}

func pdfFilesProducer(pdfFiles chan string, deltaInMins int) {
	defer close(pdfFiles)
	user, _ := user.Current()
	homeDir := user.HomeDir
	calibreDir := filepath.Join(homeDir, "Calibre Library")
	filepath.Walk(calibreDir, createWalkFunc(pdfFiles, deltaInMins))
}

func createWalkFunc(pdfFiles chan string, deltaInMins int) filepath.WalkFunc {

	return func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".pdf" {
			fileInfo, _ := os.Stat(path)
			if time.Now().Sub(fileInfo.ModTime()) < time.Duration(deltaInMins)*time.Minute {
				fmt.Printf("+ Copying %s to Dropbox\n", fileInfo.Name())
				pdfFiles <- path
			}
		}
		return nil
	}
}

func consumePdfFiles(pdfFiles chan string, destFolderName string, wg *sync.WaitGroup) {
	for src := range pdfFiles {
		user, _ := user.Current()
		outputFolder := filepath.Join(user.HomeDir, "Dropbox", destFolderName)

		err := os.Mkdir(outputFolder, 0755)
		if err != nil {
			panic(err)
		}

		srcFile, err := os.Open(src)
		if err != nil {
			panic(err)
		}

		outputFilename := filepath.Join(outputFolder, filepath.Base(src))
		err = os.Remove(outputFilename)
		if err != nil && !os.IsNotExist(err) {
			panic(err)
		}

		destFile, err := os.Create(outputFilename)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			panic(err)
		}
		fmt.Printf("* Copied: %s >>>>>> %s\n", filepath.Base(src), filepath.Dir(outputFilename))
		defer srcFile.Close()
		defer destFile.Close()
	}
	wg.Done()
}

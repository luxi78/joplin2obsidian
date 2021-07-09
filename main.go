package main

import (
	"flag"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"os"
)

var Version string

func main() {
	SrcPath = flag.String("s", "","Specify the directory where Joplin exported the RAW data" )
	DestPath = flag.String("d", "", "The directory of Obsidian vault")
	flag.Parse()

	fmt.Printf("joplin2obsidian %s\n\n", Version)

	if len(*SrcPath)==0 || len(*DestPath)==0 {
		_, err := fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		CheckError(err)
		flag.PrintDefaults()
		os.Exit(-1)
	}

	chkPath := func(p string) {
		if fi, err := os.Stat(p);os.IsNotExist(err) {
			println(fmt.Sprintf("%s isn't exist", p))
			os.Exit(-1)
		} else {
			if !fi.Mode().IsDir() {
				println(fmt.Sprintf("%s isn't a directory", p))
				os.Exit(-1)
			}
		}
	}
	chkPath(*SrcPath)
	chkPath(*DestPath)

	progress := make(chan int,1)
	done := make(chan bool, 1)
	go HandlingCoreBusiness(progress, done)

	go func() {
		var bar *progressbar.ProgressBar
		step := 0
		for newStep := range progress {
			if newStep != step {
				if bar != nil {
					bar.Finish()
				}
				step = newStep
				bar = progressbar.Default(-1, StepDesc[newStep])
				err := bar.Set(0)
				CheckError(err)
			}
			if bar!=nil {
				err := bar.Add(1)
				CheckError(err)
			}
		}
		if bar!=nil {
			bar.Finish()
		}
	}()

	<-done
	fmt.Printf("\n\nDone!\n\n")
	fmt.Println(fmt.Sprintf("The next step is to open %s as vault in Obsidian, Then you will see what you want to see.", *DestPath))

}

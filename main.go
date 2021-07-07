package main

import (
	"flag"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"os"
	"time"
)

var Version string
func testBar() {
	bar := progressbar.Default(-1, "aaa", "bbb")
	err := bar.Set(0)
	CheckError(err)
	for i := 0; i < 100; i++ {
		err = bar.Add(1)
		CheckError(err)
		time.Sleep(40 * time.Millisecond)
	}
}

func main() {
	//testBar()
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

	HandlingCoreBusiness()
	fmt.Println("Done!")

}

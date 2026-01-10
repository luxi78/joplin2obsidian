// Package main implements joplin2obsidian, a conversion tool to migrate notes from Joplin to Obsidian.
// It converts Joplin's RAW export format to Obsidian-compatible markdown with proper frontmatter
// and resource references.
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

	if err := DetectResourcesFolder(); err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting resources folder: %v\n", err)
		os.Exit(1)
	}

	stats := &ConversionStats{}
	progress := make(chan int,1)
	done := make(chan bool, 1)
	go HandlingCoreBusiness(progress, done, stats)

	go func() {
		var bar *progressbar.ProgressBar
		totalFiles := 0
		for val := range progress {
			// Negative value indicates total file count
			if val < 0 {
				totalFiles = -val
				bar = progressbar.Default(int64(totalFiles), "Processing files")
				continue
			}

			// Positive value indicates progress
			if bar != nil {
				err := bar.Add(1)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Progress bar error: %v\n", err)
				}
			}
		}
		if bar != nil {
			bar.Finish()
		}
	}()

	<-done
	fmt.Printf("\n\nConversion complete!\n\n")
	fmt.Println("Summary:")
	fmt.Printf("  %c %d articles converted\n", '\u2713', stats.ArticlesConverted)
	fmt.Printf("  %c %d folders created\n", '\u2713', stats.FoldersCreated)
	fmt.Printf("  %c %d resources copied\n", '\u2713', stats.ResourcesCopied)
	if stats.FilesSkipped > 0 {
		fmt.Printf("  %c %d files skipped (see warnings above)\n", '\u26A0', stats.FilesSkipped)
	}
	fmt.Printf("\nOutput directory: %s\n", *DestPath)
	fmt.Println("\nNext step: Open the output directory as a vault in Obsidian.")

}

package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

var SrcPath *string
var DestPath *string

const ResourcesFolder string = "resources"

func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

type FileInfo struct {
	name          string
	metaIndex     int
	metaId        string
	metaParentId  string
	metaType      int //1:Article 2:Folder 4:Resource 5:Tag
	metaFileExt   string
	metaCreatedAt string
	metaUpdatedAt string
}

func (fi FileInfo) getValidName() string {
	r := strings.NewReplacer(
		"*", ".",
		"\"", "''",
		"\\", "-",
		"/", "_",
		"<", ",",
		">", ".",
		":", ";",
		"|", "-",
		"?", "!")
	return r.Replace(fi.name)
}

type Folder struct {
	*FileInfo
	parent *Folder
}

func (f Folder) getPath() string {
	if f.parent == nil {
		return path.Join(*DestPath, f.getValidName())
	} else {
		return path.Join(f.parent.getPath(), f.getValidName())
	}
}

type Article struct {
	*FileInfo
	folder  *Folder
	content string
}

func (a Article) getPath() string {
	return fmt.Sprintf("%s.md", path.Join(a.folder.getPath(), a.getValidName()))
}
func (a Article) save() {
	filePath := a.getPath()
	dirName := path.Dir(filePath)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err := os.MkdirAll(dirName, 0755)
		CheckError(err)
	}
	// optional meta info to sort by time like Joplin
	prefix := ""
	meta := a.FileInfo
	if meta != nil && meta.metaCreatedAt != "" && meta.metaUpdatedAt != "" && meta.metaId != "" {
		prefix = fmt.Sprintf("---\ncreated: %v\nupdated: %v\njoplin_id: %v\n---\n",
			meta.metaCreatedAt, meta.metaUpdatedAt, meta.metaId,
		)
	}

	err := os.WriteFile(filePath, []byte(prefix+a.content), 0644)
	CheckError(err)

	// optionally change mtime and atime
	// 2021-07-10T02:10:03.850Z
	if meta != nil && meta.metaCreatedAt != "" && meta.metaUpdatedAt != "" {
		updatedAt, err := time.Parse(time.RFC3339, meta.metaUpdatedAt)
		CheckError(err)
		err = os.Chtimes(filePath, updatedAt, updatedAt)
		CheckError(err)
	}
}

type Resource struct {
	*FileInfo
}

func (r Resource) getFileName() string {
	var fileName string
	if /*len(r.metaFileExt) > 0*/ false {
		fileName = fmt.Sprintf("%s.%s", r.metaId, r.metaFileExt)
	} else {
		resPath := path.Join(*SrcPath, "resources")
		c, err := os.ReadDir(resPath)
		CheckError(err)
		for _, entry := range c {
			if entry.IsDir() {
				continue
			}
			if strings.Index(entry.Name(), r.metaId) >= 0 {
				fileName = entry.Name()
				break
			}
		}
	}
	return fileName
}

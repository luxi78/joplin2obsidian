package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

var SrcPath *string
var DestPath *string

const ResourcesFolder string = "resources"

func CheckError(e error) {
	if e!=nil {
		panic(e)
	}
}

type FileInfo struct {
	name string
	metaIndex int
	metaId string
	metaParentId string
	metaType int	//1:Article 2:Folder 4:Resource 5:Tag
	metaFileExt string
}
func (fi FileInfo) getValidName() string {
	r := strings.NewReplacer(
		"*", ".",
		"\"","''",
		"\\","-",
		"/","_",
		"<",",",
		">",".",
		":",";",
		"|","-",
		"?","!")
	return r.Replace(fi.name)
}

type Folder struct {
	*FileInfo
	parent *Folder
}
func (f Folder) getPath() string {
	if f.parent == nil {
		return path.Join(*DestPath,f.getValidName())
	} else {
		return path.Join(f.parent.getPath(), f.getValidName())
	}
}

type Article struct {
	*FileInfo
	folder *Folder
	content string
}
func (a Article) getPath() string{
	return fmt.Sprintf("%s.md",path.Join(a.folder.getPath(), a.getValidName()))
}
func (a Article)save() {
	filePath := a.getPath()
	dirName := path.Dir(filePath)
	if _, err := os.Stat(dirName) ; os.IsNotExist(err){
		err := os.MkdirAll(dirName, 0755)
		CheckError(err)
	}
	err := ioutil.WriteFile(filePath, []byte(a.content), 0644)
	CheckError(err)
}

type Resource struct {
	*FileInfo
}
func (r Resource) getFileName() string {
	return fmt.Sprintf("%s.%s", r.metaId, r.metaFileExt)
}

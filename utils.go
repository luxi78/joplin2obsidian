package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"

	copy2 "github.com/otiai10/copy"
)

func GetFileInfo(filePath string) (*FileInfo, *string) {
	data, err := ioutil.ReadFile(filePath)
	CheckError(err)
	strData := strings.TrimSpace(string(data))
	metaIndex := strings.LastIndex(strData, "\n\n")
	if metaIndex <= 0 {
		return nil, nil
	}

	strMeta := strData[metaIndex:]
	strMeta = fmt.Sprintf("%s\n", strMeta)

	r, _ := regexp.Compile("id: *(.*)\n")
	match := r.FindStringSubmatch(strMeta)
	if len(match) < 2 {
		return nil, nil
	}
	metaId := match[1]

	r, _ = regexp.Compile("type_: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) < 2 {
		return nil, nil
	}
	metaType, err := strconv.Atoi(match[1])
	CheckError(err)
	if 1 != metaType && 2 != metaType && 4 != metaType {
		return nil, nil
	}

	metaParentId := ""
	r, _ = regexp.Compile("parent_id: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaParentId = match[1]
	}

	metaFileExt := ""
	r, _ = regexp.Compile("file_extension: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaFileExt = match[1]
	}

	r, _ = regexp.Compile("(.*)\n")
	match = r.FindStringSubmatch(strData)
	if len(match) < 2 {
		return nil, nil
	}
	name := strings.TrimSpace(match[1])

	return &FileInfo{
		name:         name,
		metaIndex:    metaIndex,
		metaId:       metaId,
		metaType:     metaType,
		metaParentId: metaParentId,
		metaFileExt:  metaFileExt,
	}, &strData
}

var StepDesc = [5]string{
	"Initializing",
	"Extracting Metadata", //1
	"Rebuilding Folders",
	"Rebuilding Articles",
	"Saving Data",
}

func HandlingCoreBusiness(progress chan<- int, done chan<- bool) {
	folderMap := make(map[string]*Folder)
	articleMap := make(map[string]*Article)
	resMap := make(map[string]*Resource)
	c, err := ioutil.ReadDir(*SrcPath)
	CheckError(err)
	for _, entry := range c {
		if entry.IsDir() ||
			path.Ext(entry.Name()) != ".md" {
			continue
		}

		filePath := path.Join(*SrcPath, entry.Name())
		fi, rawData := GetFileInfo(filePath)
		if fi == nil {
			continue
		}
		if 2 == fi.metaType {
			folder := Folder{FileInfo: fi}
			folderMap[folder.metaId] = &folder
		} else if 1 == fi.metaType {
			content := (*rawData)[:fi.metaIndex]
			r, _ := regexp.Compile("(.*\n)")
			match := r.FindStringIndex(content)
			if len(match) == 2 {
				content = strings.TrimSpace(content[match[1]:])
			}
			article := Article{FileInfo: fi, content: content}
			articleMap[article.metaId] = &article
		} else if 4 == fi.metaType {
			resMap[fi.metaId] = &Resource{FileInfo: fi}
		}
		progress <- 1
	}
	RebuildFoldersRelationship(&folderMap, progress)
	RebuildArticlesRelationship(&articleMap, &folderMap, progress)

	err = copy2.Copy(path.Join(*SrcPath, ResourcesFolder), path.Join(*DestPath, ResourcesFolder))
	CheckError(err)

	for _, article := range articleMap {
		FixResourceRef(article, &resMap, &articleMap)
		article.save()
		progress <- 4
	}

	close(progress)
	done <- true
}

func FixResourceRef(article *Article, resMap *map[string]*Resource, articleMap *map[string]*Article) {
	content := article.content
	r, _ := regexp.Compile(`(!?)\[(.*?)]\(:/(.*?)\)`)
	matchAll := r.FindAllStringSubmatchIndex(content, -1)
	for i := len(matchAll) - 1; i >= 0; i-- {
		match := matchAll[i]
		resId := strings.Split(content[match[6]:match[7]], " ")[0]

		var resFileName string
		if res, prs := (*resMap)[resId]; prs {
			resFileName = res.getFileName()
		} else if res, prs := (*articleMap)[resId]; prs {
			resFileName = path.Join(res.folder.getRelativePath(), res.getValidName())
		} else {
			resFileName = path.Join("resources", resId) // help to find lost resource
		}
		content = fmt.Sprintf("%s[[%s]]%s", content[:match[3]], resFileName, content[match[1]:])
	}
	article.content = content
}

func RebuildFoldersRelationship(folderMap *map[string]*Folder, progress chan<- int) {
	for _, folder := range *folderMap {
		if len(folder.metaParentId) == 0 {
			continue
		}
		parent := (*folderMap)[folder.metaParentId]
		folder.parent = parent
		progress <- 2
	}
}

func RebuildArticlesRelationship(articleMap *map[string]*Article, folderMap *map[string]*Folder, progress chan<- int) {
	for _, article := range *articleMap {
		if len(article.metaParentId) == 0 {
			continue
		}
		parent := (*folderMap)[article.metaParentId]
		article.folder = parent
		progress <- 3
	}
}

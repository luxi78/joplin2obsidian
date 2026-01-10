package main

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	copy2 "github.com/otiai10/copy"
)

// Pre-compiled regular expressions for better performance
var (
	regexID          = regexp.MustCompile(`id: *(.*)\n`)
	regexType        = regexp.MustCompile(`type_: *(.*)\n`)
	regexParentID    = regexp.MustCompile(`parent_id: *(.*)\n`)
	regexFileExt     = regexp.MustCompile(`file_extension: *(.*)\n`)
	regexCreatedTime = regexp.MustCompile(`created_time: *(.*)\n`)
	regexSourceURL   = regexp.MustCompile(`source_url: *(.*)\n`)
	regexAuthor      = regexp.MustCompile(`author: *(.*)\n`)
	regexFirstLine   = regexp.MustCompile(`(.*)\n`)
	regexImgTag      = regexp.MustCompile(`<img[^>]+src=":/(.*?)"[^>]*>]?(\([^)]*\))?`)
	regexMdLink      = regexp.MustCompile(`(!?)\[(.*?)]\(:/(.*?)\)`)
)

// GetFileInfo extracts metadata and content from a Joplin markdown file.
// Returns the parsed FileInfo, the raw file content, and any error encountered.
// Returns (nil, nil, nil) for files that should be skipped (not an error condition).
func GetFileInfo(filePath string) (*FileInfo, *string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	strData := strings.TrimSpace(string(data))
	metaIndex := strings.LastIndex(strData, "\n\n")
	if metaIndex <= 0 {
		return nil, nil, nil // Not an error, just not a valid Joplin file
	}

	strMeta := strData[metaIndex:]
	strMeta = fmt.Sprintf("%s\n", strMeta)

	match := regexID.FindStringSubmatch(strMeta)
	if len(match) < 2 {
		return nil, nil, nil // Missing ID, skip file
	}
	metaId := match[1]

	match = regexType.FindStringSubmatch(strMeta)
	if len(match) < 2 {
		return nil, nil, nil // Missing type, skip file
	}
	metaType, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, nil, fmt.Errorf("invalid type_ value in %s: %w", filePath, err)
	}
	if metaType != TypeArticle && metaType != TypeFolder && metaType != TypeResource {
		return nil, nil, nil // Unsupported type, skip file
	}

	metaParentId := ""
	match = regexParentID.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaParentId = match[1]
	}

	metaFileExt := ""
	match = regexFileExt.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaFileExt = match[1]
	}

	metaCreatedTime := ""
	match = regexCreatedTime.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaCreatedTime = match[1]
	}

	metaSourceUrl := ""
	match = regexSourceURL.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaSourceUrl = match[1]
	}

	metaAuthor := ""
	match = regexAuthor.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaAuthor = match[1]
	}

	match = regexFirstLine.FindStringSubmatch(strData)
	if len(match) < 2 {
		return nil, nil, nil
	}
	name := strings.TrimSpace(match[1])

	return &FileInfo{
		name:            name,
		metaIndex:       metaIndex,
		metaId:          metaId,
		metaType:        metaType,
		metaParentId:    metaParentId,
		metaFileExt:     metaFileExt,
		metaCreatedTime: metaCreatedTime,
		metaSourceUrl:   metaSourceUrl,
		metaAuthor:      metaAuthor,
	}, &strData, nil
}

var StepDesc = [5]string{
	"Initializing",
	"Extracting Metadata", //1
	"Rebuilding Folders",
	"Rebuilding Articles",
	"Saving Data",
}

// ConversionStats tracks statistics during the conversion process
type ConversionStats struct {
	ArticlesConverted int
	FoldersCreated    int
	ResourcesCopied   int
	FilesSkipped      int
}

// HandlingCoreBusiness is the main conversion logic that processes Joplin files and converts them to Obsidian format.
// It reads all markdown files from the source directory, parses metadata, rebuilds folder/article relationships,
// copies resources, and saves converted articles to the destination directory.
func HandlingCoreBusiness(progress chan<- int, done chan<- bool, stats *ConversionStats) {
	folderMap := make(map[string]*Folder)
	articleMap := make(map[string]*Article)
	resMap := make(map[string]*Resource)
	c, err := os.ReadDir(*SrcPath)
	CheckError(err)

	// Count total markdown files for progress tracking
	totalFiles := 0
	for _, entry := range c {
		if !entry.IsDir() && path.Ext(entry.Name()) == ".md" {
			totalFiles++
		}
	}

	// Signal total count as negative number (convention for progress initialization)
	if totalFiles > 0 {
		progress <- -totalFiles
	}

	for _, entry := range c {
		if entry.IsDir() ||
			path.Ext(entry.Name()) != ".md" {
			continue
		}

		filePath := path.Join(*SrcPath, entry.Name())
		fi, rawData, err := GetFileInfo(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v (skipping)\n", err)
			stats.FilesSkipped++
			continue
		}
		if fi == nil {
			continue
		}
		if fi.metaType == TypeFolder {
			folder := Folder{FileInfo: fi}
			folderMap[folder.metaId] = &folder
			stats.FoldersCreated++
		} else if fi.metaType == TypeArticle {
			content := (*rawData)[:fi.metaIndex]
			r, _ := regexp.Compile("(.*\n)")
			match := r.FindStringIndex(content)
			if len(match) == 2 {
				content = strings.TrimSpace(content[match[1]:])
			}
			article := Article{FileInfo: fi, content: content}
			articleMap[article.metaId] = &article
		} else if fi.metaType == TypeResource {
			resMap[fi.metaId] = &Resource{FileInfo: fi}
			stats.ResourcesCopied++
		}
		progress <- 1
	}
	RebuildFoldersRelationship(&folderMap, progress)
	RebuildArticlesRelationship(&articleMap, &folderMap, progress)

	// Build resource filename map once for efficient lookups
	resourceFileMap := buildResourceFileMap()

	err = copy2.Copy(path.Join(*SrcPath, ResourcesFolder), path.Join(*DestPath, ResourcesFolder))
	CheckError(err)

	for _, article := range articleMap {
		FixResourceRef(article, &resMap, &articleMap, resourceFileMap)
		if err := article.save(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v (skipping)\n", err)
			stats.FilesSkipped++
			continue
		}
		stats.ArticlesConverted++
		progress <- 4
	}

	close(progress)
	done <- true
}

// FixResourceRef converts Joplin resource references to Obsidian wiki-link format.
// It handles both HTML img tags and markdown-style links, converting them to [[filename]] format.
func FixResourceRef(article *Article, resMap *map[string]*Resource, articleMap *map[string]*Article, resourceFileMap map[string]string) {
	content := article.content

	// Handle HTML img tags: <img ... src=":/resourceId" ...> with optional trailing ] and (url)
	imgMatchAll := regexImgTag.FindAllStringSubmatchIndex(content, -1)
	for i := len(imgMatchAll) - 1; i >= 0; i-- {
		match := imgMatchAll[i]
		resId := strings.Split(content[match[2]:match[3]], " ")[0]

		var resFileName string
		if res, prs := (*resMap)[resId]; prs {
			resFileName = res.getFileName(resourceFileMap)
		} else if res, prs := (*articleMap)[resId]; prs {
			resFileName = path.Join(res.folder.getRelativePath(), res.getValidName())
		} else {
			resFileName = path.Join("resources", resId) // help to find lost resource
		}
		content = fmt.Sprintf("%s![[%s]]%s", content[:match[0]], resFileName, content[match[1]:])
	}

	// Handle markdown links: [text](:/resourceId) or ![alt](:/resourceId)
	matchAll := regexMdLink.FindAllStringSubmatchIndex(content, -1)
	for i := len(matchAll) - 1; i >= 0; i-- {
		match := matchAll[i]
		resId := strings.Split(content[match[6]:match[7]], " ")[0]

		var resFileName string
		if res, prs := (*resMap)[resId]; prs {
			resFileName = res.getFileName(resourceFileMap)
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
		parent, exists := (*folderMap)[folder.metaParentId]
		if !exists {
			fmt.Fprintf(os.Stderr, "Warning: Folder '%s' has non-existent parent ID '%s', placing at root\n", folder.name, folder.metaParentId)
			continue
		}
		folder.parent = parent
		progress <- 2
	}
}

func RebuildArticlesRelationship(articleMap *map[string]*Article, folderMap *map[string]*Folder, progress chan<- int) {
	for _, article := range *articleMap {
		if len(article.metaParentId) == 0 {
			continue
		}
		parent, exists := (*folderMap)[article.metaParentId]
		if !exists {
			fmt.Fprintf(os.Stderr, "Warning: Article '%s' has non-existent parent ID '%s', placing at root\n", article.name, article.metaParentId)
			continue
		}
		article.folder = parent
		progress <- 3
	}
}

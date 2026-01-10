package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

// Joplin metadata type constants
const (
	TypeArticle  = 1
	TypeFolder   = 2
	TypeResource = 4
	TypeTag      = 5
)

var SrcPath *string
var DestPath *string
var ResourcesFolder string

func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

func DetectResourcesFolder() error {
	// Check for "resources" first (more common), then ".resource"
	candidates := []string{"resources", ".resource"}
	for _, candidate := range candidates {
		resPath := path.Join(*SrcPath, candidate)
		if _, err := os.Stat(resPath); err == nil {
			ResourcesFolder = candidate
			return nil
		}
	}
	// Default to "resources" if neither exists (will be created if needed)
	ResourcesFolder = "resources"
	return nil
}

type FileInfo struct {
	name            string
	metaIndex       int
	metaId          string
	metaParentId    string
	metaType        int //1:Article 2:Folder 4:Resource 5:Tag
	metaFileExt     string
	metaCreatedTime string
	metaSourceUrl   string
	metaAuthor      string
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
	return path.Join(*DestPath, f.getRelativePath())
}

func (f Folder) getRelativePath() string {
	if f.parent == nil {
		return f.getValidName()
	} else {
		return path.Join(f.parent.getRelativePath(), f.getValidName())
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

// escapeYAMLValue escapes a string for safe use in YAML frontmatter
func escapeYAMLValue(s string) string {
	if s == "" {
		return s
	}
	// Check if the string needs quoting
	needsQuoting := false
	for _, char := range s {
		switch char {
		case ':', '#', '[', ']', '{', '}', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
			needsQuoting = true
			break
		}
	}
	// Also quote if it starts with special characters
	if len(s) > 0 && (s[0] == '-' || s[0] == '?') {
		needsQuoting = true
	}

	if needsQuoting {
		// Escape any existing quotes and wrap in quotes
		escaped := strings.ReplaceAll(s, "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}
	return s
}
func (a Article) save() error {
	filePath := a.getPath()
	dirName := path.Dir(filePath)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		if err := os.MkdirAll(dirName, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirName, err)
		}
	}

	content := a.content
	// Add YAML frontmatter with created_time and source_url if available
	hasFrontmatter := false
	frontmatter := ""

	if len(a.metaCreatedTime) > 0 {
		// Parse ISO 8601 datetime string (e.g., "2023-06-11T02:17:23.252Z")
		t, err := time.Parse(time.RFC3339, a.metaCreatedTime)
		if err == nil {
			// Format as ISO 8601 without milliseconds
			dateStr := t.Format("2006-01-02T15:04:05")
			frontmatter += fmt.Sprintf("time: %s\n", dateStr)
			hasFrontmatter = true
		}
	}

	if len(a.metaSourceUrl) > 0 {
		frontmatter += fmt.Sprintf("source: %s\n", escapeYAMLValue(a.metaSourceUrl))
		hasFrontmatter = true
	}

	if len(a.metaAuthor) > 0 {
		frontmatter += fmt.Sprintf("author: %s\n", escapeYAMLValue(a.metaAuthor))
		hasFrontmatter = true
	}

	if hasFrontmatter {
		content = fmt.Sprintf("---\n%s---\n\n%s", frontmatter, content)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	return nil
}

type Resource struct {
	*FileInfo
}

// getFileName returns the filename for a resource, using the provided map for efficient lookup
func (r Resource) getFileName(resourceFileMap map[string]string) string {
	// First check the map
	if fileName, exists := resourceFileMap[r.metaId]; exists {
		return fileName
	}
	// Fallback: construct from extension if available
	if len(r.metaFileExt) > 0 {
		return fmt.Sprintf("%s.%s", r.metaId, r.metaFileExt)
	}
	// Last resort: just return the ID
	return r.metaId
}

// buildResourceFileMap builds a map of resource ID to filename by reading the resources directory once
func buildResourceFileMap() map[string]string {
	fileMap := make(map[string]string)
	resPath := path.Join(*SrcPath, ResourcesFolder)
	entries, err := os.ReadDir(resPath)
	if err != nil {
		// Resources folder doesn't exist or can't be read, return empty map
		return fileMap
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		// Extract resource ID from filename (assumes ID is before the extension or at start)
		// Try to find a match by checking if filename starts with a resource ID pattern
		// For now, we'll extract the ID as the part before the first dot or the whole name
		idPart := strings.Split(fileName, ".")[0]
		if len(idPart) > 0 {
			fileMap[idPart] = fileName
		}
	}
	return fileMap
}

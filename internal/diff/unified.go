package diff

import (
	"fmt"
	"regexp"
	"strings"
)

var reHunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

type LineKind string

const (
	LineContext LineKind = "context"
	LineAdd     LineKind = "add"
	LineDelete  LineKind = "delete"
	LineMeta    LineKind = "meta"
)

type Patch struct {
	Files []File
}

type File struct {
	OldPath string
	NewPath string
	Path    string
	Hunks   []Hunk
}

type Hunk struct {
	Header string
	Lines  []Line
}

type Line struct {
	Kind    LineKind
	Content string
	OldLine int
	NewLine int
}

func ParseUnified(raw string) (Patch, error) {
	var patch Patch
	var currentFile *File
	var currentHunk *Hunk
	var oldLine, newLine int

	appendFile := func() {
		if currentFile == nil {
			return
		}
		patch.Files = append(patch.Files, *currentFile)
		currentFile = nil
		currentHunk = nil
		oldLine = 0
		newLine = 0
	}

	for _, rawLine := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(rawLine, "diff --git "):
			appendFile()
			currentFile = &File{}
			parts := strings.Fields(rawLine)
			if len(parts) >= 4 {
				currentFile.OldPath = trimDiffPath(parts[2])
				currentFile.NewPath = trimDiffPath(parts[3])
				currentFile.Path = pickDisplayPath(currentFile.OldPath, currentFile.NewPath)
			}

		case strings.HasPrefix(rawLine, "--- "):
			if currentFile == nil {
				currentFile = &File{}
			}
			currentFile.OldPath = trimDiffPath(strings.TrimPrefix(rawLine, "--- "))
			currentFile.Path = pickDisplayPath(currentFile.OldPath, currentFile.NewPath)

		case strings.HasPrefix(rawLine, "+++ "):
			if currentFile == nil {
				currentFile = &File{}
			}
			currentFile.NewPath = trimDiffPath(strings.TrimPrefix(rawLine, "+++ "))
			currentFile.Path = pickDisplayPath(currentFile.OldPath, currentFile.NewPath)

		case strings.HasPrefix(rawLine, "@@"):
			if currentFile == nil {
				return Patch{}, fmt.Errorf("diff hunk appeared before file header")
			}
			m := reHunkHeader.FindStringSubmatch(rawLine)
			if m == nil {
				return Patch{}, fmt.Errorf("invalid hunk header %q", rawLine)
			}
			oldLine = atoiOrZero(m[1])
			newLine = atoiOrZero(m[2])
			currentFile.Hunks = append(currentFile.Hunks, Hunk{Header: rawLine})
			currentHunk = &currentFile.Hunks[len(currentFile.Hunks)-1]

		case strings.HasPrefix(rawLine, "\\ No newline at end of file"):
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, Line{Kind: LineMeta, Content: rawLine})
			}

		default:
			if currentHunk == nil {
				continue
			}

			line := Line{Content: rawLine}
			switch {
			case strings.HasPrefix(rawLine, "+") && !strings.HasPrefix(rawLine, "+++"):
				line.Kind = LineAdd
				line.NewLine = newLine
				newLine++
			case strings.HasPrefix(rawLine, "-") && !strings.HasPrefix(rawLine, "---"):
				line.Kind = LineDelete
				line.OldLine = oldLine
				oldLine++
			default:
				line.Kind = LineContext
				line.OldLine = oldLine
				line.NewLine = newLine
				oldLine++
				newLine++
			}
			currentHunk.Lines = append(currentHunk.Lines, line)
		}
	}

	appendFile()
	return patch, nil
}

func (p Patch) File(path string) (File, bool) {
	for _, file := range p.Files {
		if file.Path == path {
			return file, true
		}
	}
	return File{}, false
}

func (p Patch) Paths() []string {
	paths := make([]string, 0, len(p.Files))
	for _, file := range p.Files {
		if file.Path != "" {
			paths = append(paths, file.Path)
		}
	}
	return paths
}

func (f File) ValidLines(side string) []int {
	side = strings.ToUpper(strings.TrimSpace(side))
	lines := make([]int, 0)
	seen := map[int]struct{}{}
	for _, hunk := range f.Hunks {
		for _, line := range hunk.Lines {
			var candidate int
			switch side {
			case "LEFT":
				if line.Kind != LineDelete {
					continue
				}
				candidate = line.OldLine
			default:
				if line.Kind != LineAdd && line.Kind != LineContext {
					continue
				}
				candidate = line.NewLine
			}
			if candidate == 0 {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			lines = append(lines, candidate)
		}
	}
	return lines
}

func (f File) HasCommentLine(side string, line int) bool {
	for _, candidate := range f.ValidLines(side) {
		if candidate == line {
			return true
		}
	}
	return false
}

func trimDiffPath(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "a/")
	raw = strings.TrimPrefix(raw, "b/")
	if raw == "/dev/null" {
		return ""
	}
	return raw
}

func pickDisplayPath(oldPath, newPath string) string {
	if newPath != "" {
		return newPath
	}
	return oldPath
}

func atoiOrZero(raw string) int {
	n := 0
	for _, r := range raw {
		n *= 10
		n += int(r - '0')
	}
	return n
}

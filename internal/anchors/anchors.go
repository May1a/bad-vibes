package anchors

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/may1a/bad-vibes/internal/model"
)

var reAnchorTag = regexp.MustCompile(`^\s*(?:[#>*-]\s+)?#([A-Za-z][\w-]*)\b`)

// Merge combines locally saved anchors with tags discovered from unresolved
// thread bodies. Local anchors win when the same tag exists in both places.
func Merge(local []model.Anchor, threads []model.ReviewThread) []model.Anchor {
	merged := map[string]model.Anchor{}
	for _, anchor := range discover(threads) {
		merged[anchorKey(anchor)] = anchor
	}
	for _, anchor := range local {
		merged[anchorKey(anchor)] = anchor
	}

	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		left := merged[a]
		right := merged[b]
		switch {
		case left.Tag != right.Tag:
			return strings.Compare(left.Tag, right.Tag)
		case left.Path != right.Path:
			return strings.Compare(left.Path, right.Path)
		case left.Line != right.Line:
			return left.Line - right.Line
		default:
			return strings.Compare(left.ThreadID, right.ThreadID)
		}
	})

	anchors := make([]model.Anchor, 0, len(keys))
	for _, key := range keys {
		anchors = append(anchors, merged[key])
	}
	return anchors
}

// Resolve returns a single anchor by tag, using local anchors plus discovered
// tags from unresolved thread bodies. It returns an explicit error when a tag is
// missing or matches multiple threads.
func Resolve(local []model.Anchor, threads []model.ReviewThread, tag string) (model.Anchor, error) {
	var localMatches []model.Anchor
	for _, anchor := range local {
		if anchor.Tag == tag {
			localMatches = append(localMatches, anchor)
		}
	}
	if len(localMatches) > 0 {
		return resolveMatches(tag, localMatches)
	}

	var matches []model.Anchor
	for _, anchor := range discover(threads) {
		if anchor.Tag == tag {
			matches = append(matches, anchor)
		}
	}
	return resolveMatches(tag, matches)
}

func resolveMatches(tag string, matches []model.Anchor) (model.Anchor, error) {
	switch len(matches) {
	case 0:
		return model.Anchor{}, fmt.Errorf("no anchor #%s found", tag)
	case 1:
		return matches[0], nil
	default:
		locations := make([]string, 0, len(matches))
		for _, match := range matches {
			locations = append(locations, formatLocation(match))
		}
		return model.Anchor{}, fmt.Errorf("anchor #%s matches multiple threads: %s", tag, strings.Join(locations, ", "))
	}
}

func discover(threads []model.ReviewThread) []model.Anchor {
	var anchors []model.Anchor

	for _, thread := range threads {
		if thread.IsResolved {
			continue
		}

		seenInThread := map[string]struct{}{}
		for _, comment := range thread.Comments {
			for _, tag := range extractTags(comment.Body) {
				if _, ok := seenInThread[tag]; ok {
					continue
				}
				seenInThread[tag] = struct{}{}

				anchors = append(anchors, model.Anchor{
					Tag:      tag,
					Path:     thread.Path,
					Line:     threadLine(thread),
					Body:     comment.Body,
					Created:  comment.CreatedAt,
					ThreadID: thread.ID,
				})
			}
		}
	}
	return anchors
}

func extractTags(body string) []string {
	tags := []string{}
	seen := map[string]struct{}{}
	for line := range strings.SplitSeq(body, "\n") {
		match := reAnchorTag.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		tag := match[1]
		if strings.EqualFold(tag, "PR") {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func threadLine(thread model.ReviewThread) int {
	if thread.Line > 0 {
		return thread.Line
	}
	return thread.StartLine
}

func anchorKey(anchor model.Anchor) string {
	path := anchor.Path
	if path == "" {
		path = "@pr"
	}
	threadID := anchor.ThreadID
	if threadID == "" {
		threadID = path + ":" + strconv.Itoa(anchor.Line)
	}
	return anchor.Tag + "|" + threadID
}

func formatLocation(anchor model.Anchor) string {
	if anchor.Path == "" {
		return "PR-level comment"
	}
	if anchor.Line > 0 {
		return anchor.Path + ":" + strconv.Itoa(anchor.Line)
	}
	return anchor.Path
}

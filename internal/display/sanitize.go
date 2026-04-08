package display

import (
	"regexp"
	"strings"
)

const codeRabbitBot = "coderabbitai[bot]"

func isCodeRabbit(author string) bool {
	return author == codeRabbitBot || author == "coderabbitai"
}

var (
	reHTMLComment   = regexp.MustCompile(`(?s)<!--.*?-->`)
	reDetailsBlock  = regexp.MustCompile(`(?s)<details>.*?</details>`)
	reCollapseBlank = regexp.MustCompile(`\n{3,}`)
	reSeverityLine  = regexp.MustCompile(`^_(.+?)_$`)
	reTwoBadgeLine  = regexp.MustCompile(`^_(.+?)_\s*\|\s*_(.+?)_$`)
	reBoldTitle     = regexp.MustCompile(`^\*\*(.+?)\*\*$`)
	reShareLink     = regexp.MustCompile(`(?m)^\[.*?\]\(https://coderabbit\.ai.*?\)$`)
)

func SanitizeCodeRabbitBody(body string) string {
	body = reHTMLComment.ReplaceAllLiteralString(body, "")
	body = reDetailsBlock.ReplaceAllLiteralString(body, "")
	body = reShareLink.ReplaceAllLiteralString(body, "")
	body = stripPromoFooter(body)
	body = reCollapseBlank.ReplaceAllString(body, "\n\n")
	return strings.TrimSpace(body)
}

func stripPromoFooter(body string) string {
	idx := strings.Index(body, "\n---\n")
	if idx < 0 {
		return body
	}
	candidate := body[idx+5:]
	lower := strings.ToLower(candidate)
	if strings.Contains(lower, "coderabbit") || strings.Contains(lower, "share") {
		return body[:idx]
	}
	return body
}

type codeRabbitSummary struct {
	Severity    string
	Title       string
	Description string
}

func ExtractCodeRabbitSummary(body string) codeRabbitSummary {
	var severity, title, description string

	lines := strings.Split(body, "\n")
	var descLines []string
	foundTitle := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if severity == "" {
			if m := reTwoBadgeLine.FindStringSubmatch(trimmed); m != nil {
				severity = m[1] + " | " + m[2]
				continue
			}
			if m := reSeverityLine.FindStringSubmatch(trimmed); m != nil {
				severity = m[1]
				continue
			}
		}

		if title == "" && !foundTitle {
			if m := reBoldTitle.FindStringSubmatch(trimmed); m != nil {
				title = m[1]
				foundTitle = true
				continue
			}
		}

		if strings.HasPrefix(trimmed, "<details") || strings.HasPrefix(trimmed, "<!--") {
			break
		}

		if foundTitle && trimmed != "" {
			descLines = append(descLines, trimmed)
		}
	}

	if len(descLines) > 0 {
		description = descLines[0]
	}

	return codeRabbitSummary{
		Severity:    severity,
		Title:       title,
		Description: description,
	}
}

func codeRabbitSeverityStyle(severity string) string {
	s := strings.ToLower(severity)
	switch {
	case strings.Contains(s, "major"), strings.Contains(s, "potential issue"), strings.Contains(s, "critical"):
		return "[🟠 Major]"
	case strings.Contains(s, "minor"):
		return "[🟡 Minor]"
	case strings.Contains(s, "nitpick"):
		return "[✅ Nitpick]"
	default:
		return "[" + severity + "]"
	}
}

func codeRabbitCompactBody(body string) string {
	clean := SanitizeCodeRabbitBody(body)
	summary := ExtractCodeRabbitSummary(clean)
	parts := []string{}
	if summary.Severity != "" {
		parts = append(parts, codeRabbitSeverityStyle(summary.Severity))
	}
	if summary.Title != "" {
		parts = append(parts, summary.Title)
	}
	if summary.Description != "" {
		parts = append(parts, summary.Description)
	}
	compact := strings.Join(parts, " ")
	if strings.TrimSpace(compact) != "" {
		return compact
	}
	return previewBody(clean)
}

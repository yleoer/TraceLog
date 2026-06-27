package jira

import (
	"fmt"
	"strings"
)

func ADFToMarkdown(value any) string {
	if value == nil {
		return ""
	}
	root, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(renderADFNode(root, 0))
}

func renderADFNode(node map[string]any, depth int) string {
	nodeType, _ := node["type"].(string)
	content := renderADFContent(node["content"], depth)
	switch nodeType {
	case "doc":
		return content
	case "paragraph":
		return strings.TrimSpace(content) + "\n\n"
	case "heading":
		level := intFromAttrs(node, "level", 2)
		if level < 1 || level > 6 {
			level = 2
		}
		return strings.Repeat("#", level) + " " + strings.TrimSpace(content) + "\n\n"
	case "bulletList":
		return content + "\n"
	case "orderedList":
		return content + "\n"
	case "listItem":
		prefix := "- "
		if depth > 0 {
			prefix = strings.Repeat("  ", depth) + "- "
		}
		return prefix + strings.TrimSpace(strings.ReplaceAll(content, "\n\n", "\n")) + "\n"
	case "blockquote":
		lines := strings.Split(strings.TrimSpace(content), "\n")
		for i, line := range lines {
			lines[i] = "> " + line
		}
		return strings.Join(lines, "\n") + "\n\n"
	case "codeBlock":
		return "```\n" + strings.TrimSpace(content) + "\n```\n\n"
	case "rule":
		return "---\n\n"
	case "hardBreak":
		return "\n"
	case "text":
		text, _ := node["text"].(string)
		return applyMarks(text, node["marks"])
	case "inlineCard":
		if attrs, ok := node["attrs"].(map[string]any); ok {
			if url, ok := attrs["url"].(string); ok {
				return url
			}
		}
		return ""
	case "mention":
		if attrs, ok := node["attrs"].(map[string]any); ok {
			if text, ok := attrs["text"].(string); ok {
				return text
			}
		}
		return ""
	default:
		return content
	}
}

func renderADFContent(value any, depth int) string {
	items, ok := value.([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, item := range items {
		node, ok := item.(map[string]any)
		if !ok {
			continue
		}
		nodeDepth := depth
		if nodeType, _ := node["type"].(string); nodeType == "listItem" {
			nodeDepth = depth + 1
		}
		b.WriteString(renderADFNode(node, nodeDepth))
	}
	return b.String()
}

func applyMarks(text string, marksValue any) string {
	marks, ok := marksValue.([]any)
	if !ok {
		return text
	}
	for _, markValue := range marks {
		mark, ok := markValue.(map[string]any)
		if !ok {
			continue
		}
		markType, _ := mark["type"].(string)
		switch markType {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "link":
			if attrs, ok := mark["attrs"].(map[string]any); ok {
				if href, ok := attrs["href"].(string); ok {
					text = fmt.Sprintf("[%s](%s)", text, href)
				}
			}
		}
	}
	return text
}

func intFromAttrs(node map[string]any, key string, fallback int) int {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return fallback
	}
	value, ok := attrs[key].(float64)
	if !ok {
		return fallback
	}
	return int(value)
}

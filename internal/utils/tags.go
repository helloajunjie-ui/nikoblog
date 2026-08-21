package utils

import (
	"regexp"
	"strings"
)

// tagRegex 匹配 #标签，标签由中文、字母、数字、下划线组成
var tagRegex = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)

// urlRegex 匹配 Markdown 链接 [text](url) 与裸 URL，用于剔除其中的 #锚点，避免被误判为标签
var urlRegex = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)|https?://[^\s\)\]]+`)

// ExtractTags 从文本中正则解析 #标签，返回去重后的标签列表。
// 先剔除 Markdown 链接与裸 URL，避免 URL 中的 #锚点（如 #reply0）被误判为标签。
func ExtractTags(content string) []string {
	content = urlRegex.ReplaceAllString(content, "")
	matches := tagRegex.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		tags = append(tags, name)
	}
	return tags
}

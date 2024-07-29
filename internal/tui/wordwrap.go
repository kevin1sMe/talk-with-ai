package tui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// WrapWords 使用mattn/go-runewidth库来精确计算字符宽度
func WrapWords(s string, maxWidth int) string {
	var builder strings.Builder
	currentWidth := 0
	for _, r := range s {
		charWidth := runewidth.RuneWidth(r)
		if currentWidth+charWidth > maxWidth {
			builder.WriteString("\n")
			currentWidth = 0
		}
		builder.WriteRune(r)
		currentWidth += charWidth
	}
	return builder.String()
}

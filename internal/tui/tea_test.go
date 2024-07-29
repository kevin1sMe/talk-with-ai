package tui

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"golang.org/x/text/width"
)

// getRuneWidth 获取字符的显示宽度
func getRuneWidth(r rune) int {
	properties := width.LookupRune(r)
	switch properties.Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	case width.Neutral, width.EastAsianNarrow, width.EastAsianHalfwidth:
		return 1
	default:
		return 1
	}
}

// wrapChinese2 使用mattn/go-runewidth库来精确计算字符宽度
func wrapChinese2(s string, maxWidth int) string {
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

// wrapChineseImproved 使用golang.org/x/text/width库来精确计算字符宽度
func wrapChineseImproved(s string, maxWidth int) string {
	var builder strings.Builder
	currentWidth := 0
	for _, r := range s {
		charWidth := getRuneWidth(r)
		if currentWidth+charWidth > maxWidth {
			builder.WriteString("\n")
			currentWidth = 0
		}
		builder.WriteRune(r)
		currentWidth += charWidth
	}
	return builder.String()
}

func TestWordWrap(t *testing.T) {
	s := `《西游记》是中国古典四大名著之一，abcdef由明代作家吴承恩创作。小说12345讲述了唐僧师徒四人——唐僧、孙悟>空、猪八戒、沙僧，为了取回佛教经典，历经九九八十一难，跋涉十万八千里，最终到达西天极乐世界的冒险故事。孙悟空以其七十二变和火眼金睛，成为了智慧与勇气的化身；猪八戒憨厚可爱，沙僧忠诚可靠。这部小说融合了神话、幻想、现实，反映了人性的多面和真善美的追求。它不仅是一部文学杰作，也深刻揭示了当时社会的各种矛盾和问题，影响深远，历久弥新。`
	// t.Log(wordwrap.String(s, 10))
	t.Log("-----------------")
	wrapped := wrapChineseImproved(s, 10)
	t.Log(wrapped)
	t.Log("-----------------")
	wrapped = wrapChinese2(s, 10)
	t.Log(wrapped)

}

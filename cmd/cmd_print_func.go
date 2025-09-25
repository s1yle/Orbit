package cmd

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// BoxStyle 定义边框样式
type BoxStyle struct {
	Horizontal string
	Vertical   string
	Corner     string
	Padding    int
}

// 预定义几种边框样式
var (
	SimpleStyle = BoxStyle{"-", "|", "+", 2}
	BoldStyle   = BoxStyle{"=", "‖", "#", 2}
	DoubleStyle = BoxStyle{"=", "║", "╔╗╚╝", 2}
)

// PrintBoxedText 输出带边框的文本
func PrintBoxedText(content string, style BoxStyle) {
	PrintBoxedTextWithTitle(content, "", style)
}

func PrintBoxedTextWithTitle(content, title string, style BoxStyle) {
	fmt.Println()
	// 计算实际显示宽度
	contentWidth := runewidth.StringWidth(content)
	titleWidth := runewidth.StringWidth(title)

	// 确定框的宽度
	boxWidth := max(contentWidth, titleWidth) + style.Padding*2
	if boxWidth < 20 {
		boxWidth = 20
	}

	// 构建边框
	topBorder := style.Corner + strings.Repeat(style.Horizontal, boxWidth) + style.Corner
	bottomBorder := style.Corner + strings.Repeat(style.Horizontal, boxWidth) + style.Corner
	emptyLine := style.Vertical + strings.Repeat(" ", boxWidth) + style.Vertical

	fmt.Println(topBorder)
	fmt.Println(emptyLine)

	// 输出标题（居中）
	if title != "" {
		titlePadding := (boxWidth - titleWidth) / 2
		leftSpaces := strings.Repeat(" ", titlePadding)
		rightSpaces := strings.Repeat(" ", boxWidth-titleWidth-titlePadding)
		fmt.Printf("%s%s%s%s%s\n", style.Vertical, leftSpaces, title, rightSpaces, style.Vertical)
		fmt.Println(emptyLine)
	}

	// 输出内容（居中）
	contentPadding := (boxWidth - contentWidth) / 2
	leftSpaces := strings.Repeat(" ", contentPadding)
	rightSpaces := strings.Repeat(" ", boxWidth-contentWidth-contentPadding)
	fmt.Printf("%s%s%s%s%s\n", style.Vertical, leftSpaces, content, rightSpaces, style.Vertical)

	fmt.Println(emptyLine)
	fmt.Println(bottomBorder)
	fmt.Println()
}

// 简化函数：用于测试日志
func LogBoxedTest(t testing.TB, content string) {
	t.Logf("\n%s", generateBox(content))
}

// 生成边框字符串（不直接输出）
func generateBox(content string) string {
	width := utf8.RuneCountInString(content) + 4 // 基础边距
	if width < 30 {
		width = 30
	}

	var sb strings.Builder
	border := "+" + strings.Repeat("-", width) + "+"

	sb.WriteString(border + "\n")
	sb.WriteString("|" + strings.Repeat(" ", width) + "|\n")

	// 内容居中
	padding := (width - utf8.RuneCountInString(content)) / 2
	leftSpaces := strings.Repeat(" ", padding)
	rightSpaces := strings.Repeat(" ", width-padding-utf8.RuneCountInString(content))
	sb.WriteString("|" + leftSpaces + content + rightSpaces + "|\n")

	sb.WriteString("|" + strings.Repeat(" ", width) + "|\n")
	sb.WriteString(border)

	return sb.String()
}

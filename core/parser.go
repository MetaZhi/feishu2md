package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Wsine/feishu2md/utils"
	"github.com/olekukonko/tablewriter"
)

type Parser struct {
	useHTMLTags bool
	Assets      []AssetRef
	blockMap    map[string]*Block
}

func NewParser(config OutputConfig) *Parser {
	return &Parser{
		useHTMLTags: config.UseHTMLTags,
		Assets:      make([]AssetRef, 0),
		blockMap:    make(map[string]*Block),
	}
}

var docxCodeLang2MdStr = map[int]string{
	1:  "",
	2:  "abap",
	3:  "ada",
	4:  "apache",
	5:  "apex",
	6:  "assembly",
	7:  "bash",
	8:  "csharp",
	9:  "cpp",
	10: "c",
	11: "cobol",
	12: "css",
	13: "coffeescript",
	14: "d",
	15: "dart",
	16: "delphi",
	17: "django",
	18: "dockerfile",
	19: "erlang",
	20: "fortran",
	21: "foxpro",
	22: "go",
	23: "groovy",
	24: "html",
	25: "htmlbars",
	26: "http",
	27: "haskell",
	28: "json",
	29: "java",
	30: "javascript",
	31: "julia",
	32: "kotlin",
	33: "latex",
	34: "lisp",
	35: "logo",
	36: "lua",
	37: "matlab",
	38: "makefile",
	39: "markdown",
	40: "nginx",
	41: "objectivec",
	42: "openedge-abl",
	43: "php",
	44: "perl",
	45: "postscript",
	46: "powershell",
	47: "prolog",
	48: "protobuf",
	49: "python",
	50: "r",
	51: "rpg",
	52: "ruby",
	53: "rust",
	54: "sas",
	55: "scss",
	56: "sql",
	57: "scala",
	58: "scheme",
	59: "scratch",
	60: "shell",
	61: "swift",
	62: "thrift",
	63: "typescript",
	64: "vbscript",
	65: "vbnet",
	66: "xml",
	67: "yaml",
}

func renderMarkdownTable(data [][]string) string {
	builder := &strings.Builder{}
	table := tablewriter.NewWriter(builder)
	table.SetCenterSeparator("|")
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetAutoMergeCells(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetHeader(data[0])
	table.AppendBulk(data[1:])
	table.Render()
	return builder.String()
}

func (p *Parser) ParseDocxContent(doc *Document, blocks []*Block) string {
	for _, block := range blocks {
		p.blockMap[block.BlockID] = block
	}
	entryBlock := p.blockMap[doc.DocumentID]
	return p.parseBlock(entryBlock, 0)
}

func (p *Parser) parseBlock(block *Block, indentLevel int) string {
	if block == nil {
		return ""
	}
	// IndentationLevel in the block's style overrides the hierarchy-derived level.
	// This handles the flat-sibling layout Feishu uses for nested lists.
	effectiveLevel := indentLevel
	switch block.BlockType {
	case BlockTypeBullet:
		if block.Bullet != nil && block.Bullet.Style != nil {
			effectiveLevel = parseIndentationLevel(block.Bullet.Style.IndentationLevel, indentLevel)
		}
	case BlockTypeOrdered:
		if block.Ordered != nil && block.Ordered.Style != nil {
			effectiveLevel = parseIndentationLevel(block.Ordered.Style.IndentationLevel, indentLevel)
		}
	}
	buf := new(strings.Builder)
	buf.WriteString(strings.Repeat("    ", effectiveLevel))
	switch block.BlockType {
	case BlockTypePage:
		buf.WriteString(p.parsePage(block))
	case BlockTypeText:
		buf.WriteString(p.parseTextBlock(block.Text))
		buf.WriteString(p.parseChildSequence(block.Children, indentLevel+1))
	case BlockTypeCallout:
		buf.WriteString(p.parseCallout(block))
	case BlockTypeHeading1:
		buf.WriteString(p.parseHeading(block, 1))
	case BlockTypeHeading2:
		buf.WriteString(p.parseHeading(block, 2))
	case BlockTypeHeading3:
		buf.WriteString(p.parseHeading(block, 3))
	case BlockTypeHeading4:
		buf.WriteString(p.parseHeading(block, 4))
	case BlockTypeHeading5:
		buf.WriteString(p.parseHeading(block, 5))
	case BlockTypeHeading6:
		buf.WriteString(p.parseHeading(block, 6))
	case BlockTypeHeading7:
		buf.WriteString(p.parseHeading(block, 7))
	case BlockTypeHeading8:
		buf.WriteString(p.parseHeading(block, 8))
	case BlockTypeHeading9:
		buf.WriteString(p.parseHeading(block, 9))
	case BlockTypeBullet:
		buf.WriteString(p.parseBullet(block, effectiveLevel))
	case BlockTypeOrdered:
		buf.WriteString(p.parseOrdered(block, effectiveLevel))
	case BlockTypeCode:
		buf.WriteString(p.parseCode(block.Code))
	case BlockTypeQuote:
		buf.WriteString("> ")
		buf.WriteString(p.parseTextBlock(block.Quote))
	case BlockTypeEquation:
		buf.WriteString("$$\n")
		buf.WriteString(p.parseTextBlock(block.Equation))
		buf.WriteString("\n$$\n")
	case BlockTypeTodo:
		buf.WriteString(p.parseTodo(block.Todo))
	case BlockTypeDivider:
		buf.WriteString("---\n")
	case BlockTypeImage:
		buf.WriteString(p.parseAsset(AssetKindImage, block.Image.Token))
	case BlockTypeTableCell:
		buf.WriteString(p.parseTableCell(block))
	case BlockTypeTable:
		buf.WriteString(p.parseTable(block.Table))
	case BlockTypeQuoteContainer:
		buf.WriteString(p.parseQuoteContainer(block))
	case BlockTypeGrid:
		buf.WriteString(p.parseGrid(block, indentLevel))
	default:
		if block.Board != nil {
			buf.WriteString(p.parseAsset(AssetKindWhiteboard, block.Board.Token))
		}
	}
	return buf.String()
}

func (p *Parser) parsePage(block *Block) string {
	buf := new(strings.Builder)
	buf.WriteString("# ")
	buf.WriteString(p.parseTextBlock(block.Page))
	buf.WriteString("\n")
	buf.WriteString(p.parseChildSequence(block.Children, 0))
	return buf.String()
}

func (p *Parser) parseTextBlock(block *TextBlock) string {
	if block == nil {
		return "\n"
	}
	buf := new(strings.Builder)
	inline := len(block.Elements) > 1
	for _, element := range block.Elements {
		buf.WriteString(p.parseTextElement(element, inline))
	}
	buf.WriteString("\n")
	return buf.String()
}

func (p *Parser) parseCallout(block *Block) string {
	buf := new(strings.Builder)
	buf.WriteString(">[!TIP] \n")
	buf.WriteString(p.parseChildSequence(block.Children, 0))
	return buf.String()
}

func (p *Parser) parseTextElement(element *TextElement, inline bool) string {
	if element == nil {
		return ""
	}
	buf := new(strings.Builder)
	if element.TextRun != nil {
		buf.WriteString(p.parseTextRun(element.TextRun))
	}
	if element.MentionUser != nil {
		buf.WriteString(element.MentionUser.UserID)
	}
	if element.MentionDoc != nil {
		buf.WriteString(fmt.Sprintf(
			"[%s](%s)",
			element.MentionDoc.Title,
			utils.UnescapeURL(element.MentionDoc.URL),
		))
	}
	if element.Equation != nil {
		symbol := "$$"
		if inline {
			symbol = "$"
		}
		buf.WriteString(symbol + strings.TrimSuffix(element.Equation.Content, "\n") + symbol)
	}
	return buf.String()
}

func (p *Parser) parseTextRun(run *TextRun) string {
	if run == nil {
		return ""
	}
	buf := new(strings.Builder)
	postWrite := ""
	style := run.TextElementStyle
	if style != nil {
		switch {
		case style.Bold:
			postWrite = p.wrapStyle(buf, "**", "**", "<strong>", "</strong>")
		case style.Italic:
			postWrite = p.wrapStyle(buf, "_", "_", "<em>", "</em>")
		case style.Strikethrough:
			postWrite = p.wrapStyle(buf, "~~", "~~", "<del>", "</del>")
		case style.Underline:
			buf.WriteString("<u>")
			postWrite = "</u>"
		case style.InlineCode:
			buf.WriteString("`")
			postWrite = "`"
		case style.Link != nil:
			buf.WriteString("[")
			postWrite = fmt.Sprintf("](%s)", utils.UnescapeURL(style.Link.URL))
		}
	}
	buf.WriteString(run.Content)
	buf.WriteString(postWrite)
	return buf.String()
}

func (p *Parser) wrapStyle(
	buf *strings.Builder,
	mdOpen, mdClose, htmlOpen, htmlClose string,
) string {
	if p.useHTMLTags {
		buf.WriteString(htmlOpen)
		return htmlClose
	}
	buf.WriteString(mdOpen)
	return mdClose
}

func (p *Parser) parseHeading(block *Block, headingLevel int) string {
	buf := new(strings.Builder)
	buf.WriteString(strings.Repeat("#", headingLevel))
	buf.WriteString(" ")
	field := reflect.ValueOf(block).Elem().FieldByName(fmt.Sprintf("Heading%d", headingLevel))
	buf.WriteString(p.parseTextBlock(field.Interface().(*TextBlock)))
	buf.WriteString(p.parseChildSequence(block.Children, 0))
	return buf.String()
}

func (p *Parser) parseAsset(kind AssetKind, token string) string {
	if token == "" {
		return ""
	}
	p.Assets = append(p.Assets, AssetRef{Kind: kind, Token: token})
	return fmt.Sprintf("![](%s)\n", token)
}

func parseIndentationLevel(s string, defaultLevel int) int {
	if s == "" {
		return defaultLevel
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultLevel
}

func (p *Parser) parseBullet(block *Block, indentLevel int) string {
	buf := new(strings.Builder)
	buf.WriteString("- ")
	buf.WriteString(p.parseTextBlock(block.Bullet))
	if p.startsWithList(block.Children) {
		buf.WriteString("\n")
	}
	buf.WriteString(p.parseChildSequence(block.Children, indentLevel+1))
	return buf.String()
}

func (p *Parser) parseOrdered(block *Block, indentLevel int) string {
	buf := new(strings.Builder)
	buf.WriteString(fmt.Sprintf("%d. ", p.orderedIndex(block)))
	buf.WriteString(p.parseTextBlock(block.Ordered))
	if p.startsWithList(block.Children) {
		buf.WriteString("\n")
	}
	buf.WriteString(p.parseChildSequence(block.Children, indentLevel+1))
	return buf.String()
}

func (p *Parser) orderedIndex(block *Block) int {
	if block == nil || block.Ordered == nil || block.Ordered.Style == nil {
		return 1
	}
	sequence := block.Ordered.Style.Sequence
	if sequence != "" && sequence != "auto" {
		if value, err := strconv.Atoi(sequence); err == nil && value > 0 {
			return value
		}
	}
	parent := p.blockMap[block.ParentID]
	if parent == nil {
		return 1
	}
	currentIndent := p.listIndent(block)
	for index, childID := range parent.Children {
		if childID != block.BlockID {
			continue
		}
		order := 1
		for previous := index - 1; previous >= 0; previous-- {
			sibling := p.blockMap[parent.Children[previous]]
			if sibling == nil {
				continue
			}
			if sibling.BlockType != BlockTypeOrdered {
				break
			}
			siblingIndent := p.listIndent(sibling)
			if siblingIndent < currentIndent {
				break
			}
			if siblingIndent == currentIndent {
				order++
			}
		}
		return order
	}
	return 1
}

func (p *Parser) parseCode(block *TextBlock) string {
	buf := new(strings.Builder)
	language := ""
	if block != nil && block.Style != nil {
		language = docxCodeLang2MdStr[block.Style.Language]
	}
	buf.WriteString("```" + language + "\n")
	buf.WriteString(strings.TrimSpace(p.parseTextBlock(block)))
	buf.WriteString("\n```\n")
	return buf.String()
}

func (p *Parser) parseTodo(block *TextBlock) string {
	buf := new(strings.Builder)
	done := block != nil && block.Style != nil && block.Style.Done
	if done {
		buf.WriteString("- [x] ")
	} else {
		buf.WriteString("- [ ] ")
	}
	buf.WriteString(p.parseTextBlock(block))
	return buf.String()
}

func (p *Parser) parseTableCell(block *Block) string {
	buf := new(strings.Builder)
	buf.WriteString(strings.ReplaceAll(strings.TrimSuffix(p.parseChildSequence(block.Children, 0), "\n"), "\n", "<br/>"))
	return buf.String()
}

func (p *Parser) parseTable(table *TableBlock) string {
	if table == nil || table.Property == nil || table.Property.ColumnSize == 0 {
		return ""
	}
	hasMerge := false
	mergeInfoMap := make(map[int]map[int]*TableMergeInfo)
	for index, merge := range table.Property.MergeInfo {
		if merge == nil {
			continue
		}
		if merge.RowSpan > 1 || merge.ColSpan > 1 {
			hasMerge = true
		}
		rowIndex := index / table.Property.ColumnSize
		colIndex := index % table.Property.ColumnSize
		if mergeInfoMap[rowIndex] == nil {
			mergeInfoMap[rowIndex] = make(map[int]*TableMergeInfo)
		}
		mergeInfoMap[rowIndex][colIndex] = merge
	}
	rows := make([][]string, 0)
	for index, blockID := range table.Cells {
		cell := strings.ReplaceAll(p.parseBlock(p.blockMap[blockID], 0), "\n", "")
		if !hasMerge {
			cell = strings.ReplaceAll(cell, "<br/>", "")
		}
		rowIndex := index / table.Property.ColumnSize
		colIndex := index % table.Property.ColumnSize
		for len(rows) <= rowIndex {
			rows = append(rows, []string{})
		}
		for len(rows[rowIndex]) <= colIndex {
			rows[rowIndex] = append(rows[rowIndex], "")
		}
		rows[rowIndex][colIndex] = cell
	}
	if hasMerge {
		return p.renderHTMLTable(rows, mergeInfoMap)
	}
	return renderMarkdownTable(rows)
}

func (p *Parser) renderHTMLTable(rows [][]string, mergeInfoMap map[int]map[int]*TableMergeInfo) string {
	buf := new(strings.Builder)
	buf.WriteString("<table>\n")
	processed := make(map[string]bool)
	for rowIndex, row := range rows {
		buf.WriteString("<tr>\n")
		for colIndex, content := range row {
			key := fmt.Sprintf("%d-%d", rowIndex, colIndex)
			if processed[key] {
				continue
			}
			merge := mergeInfoMap[rowIndex][colIndex]
			if merge == nil {
				buf.WriteString(fmt.Sprintf("<td>%s</td>", content))
				continue
			}
			buf.WriteString(fmt.Sprintf("<td%s>%s</td>", tableCellAttrs(merge), content))
			for r := rowIndex; r < rowIndex+merge.RowSpan; r++ {
				for c := colIndex; c < colIndex+merge.ColSpan; c++ {
					processed[fmt.Sprintf("%d-%d", r, c)] = true
				}
			}
		}
		buf.WriteString("</tr>\n")
	}
	buf.WriteString("</table>\n")
	return buf.String()
}

func tableCellAttrs(merge *TableMergeInfo) string {
	attrs := ""
	if merge.RowSpan > 1 {
		attrs += fmt.Sprintf(` rowspan="%d"`, merge.RowSpan)
	}
	if merge.ColSpan > 1 {
		attrs += fmt.Sprintf(` colspan="%d"`, merge.ColSpan)
	}
	return attrs
}

func (p *Parser) parseQuoteContainer(block *Block) string {
	buf := new(strings.Builder)
	for _, line := range strings.Split(strings.TrimSuffix(p.parseChildSequence(block.Children, 0), "\n"), "\n") {
		if line == "" {
			continue
		}
		buf.WriteString("> ")
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	return buf.String()
}

func (p *Parser) parseGrid(block *Block, indentLevel int) string {
	buf := new(strings.Builder)
	for _, childID := range block.Children {
		columnBlock := p.blockMap[childID]
		if columnBlock == nil {
			continue
		}
		buf.WriteString(p.parseChildSequence(columnBlock.Children, indentLevel))
	}
	return buf.String()
}

func (p *Parser) parseChildSequence(childIDs []string, indentLevel int) string {
	buf := new(strings.Builder)
	for index := 0; index < len(childIDs); {
		block := p.blockMap[childIDs[index]]
		if p.isListBlock(block) {
			rendered, next := p.parseListSequence(childIDs, index, indentLevel)
			buf.WriteString(rendered)
			index = next
			continue
		}
		buf.WriteString(p.parseBlock(block, indentLevel))
		if !strings.HasSuffix(buf.String(), "\n") {
			buf.WriteString("\n")
		}
		index++
	}
	return buf.String()
}

func (p *Parser) parseListSequence(childIDs []string, index, indentLevel int) (string, int) {
	block := p.blockMap[childIDs[index]]
	if block == nil {
		return "", index + 1
	}
	buf := new(strings.Builder)
	buf.WriteString(p.parseBlock(block, indentLevel))
	next := index + 1
	currentIndent := p.listIndent(block)
	addedSeparator := false
	for next < len(childIDs) {
		sibling := p.blockMap[childIDs[next]]
		if !p.isListBlock(sibling) {
			break
		}
		siblingIndent := p.listIndent(sibling)
		if siblingIndent <= currentIndent {
			break
		}
		if !addedSeparator {
			buf.WriteString("\n")
			addedSeparator = true
		}
		rendered, consumed := p.parseListSequence(childIDs, next, indentLevel+1)
		buf.WriteString(rendered)
		next = consumed
	}
	return buf.String(), next
}

func (p *Parser) isListBlock(block *Block) bool {
	if block == nil {
		return false
	}
	return block.BlockType == BlockTypeBullet || block.BlockType == BlockTypeOrdered
}

func (p *Parser) listIndent(block *Block) int {
	text := p.listText(block)
	if text == nil || text.Style == nil {
		return 0
	}
	return parseIndentationLevel(text.Style.IndentationLevel, 0)
}

func (p *Parser) listText(block *Block) *TextBlock {
	if block == nil {
		return nil
	}
	if block.Bullet != nil {
		return block.Bullet
	}
	return block.Ordered
}

func (p *Parser) startsWithList(childIDs []string) bool {
	if len(childIDs) == 0 {
		return false
	}
	return p.isListBlock(p.blockMap[childIDs[0]])
}

package core

import (
	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
)

func convertDocument(doc *larkdocx.Document) *Document {
	if doc == nil {
		return nil
	}
	return &Document{
		DocumentID: valueOf(doc.DocumentId),
		RevisionID: intValueOf(doc.RevisionId),
		Title:      valueOf(doc.Title),
	}
}

func convertBlocks(blocks []*larkdocx.Block) []*Block {
	result := make([]*Block, 0, len(blocks))
	for _, block := range blocks {
		result = append(result, convertBlock(block))
	}
	return result
}

func convertBlock(block *larkdocx.Block) *Block {
	if block == nil {
		return nil
	}
	return &Block{
		BlockID:        valueOf(block.BlockId),
		ParentID:       valueOf(block.ParentId),
		Children:       block.Children,
		BlockType:      intValueOf(block.BlockType),
		Page:           convertTextBlock(block.Page),
		Text:           convertTextBlock(block.Text),
		Heading1:       convertTextBlock(block.Heading1),
		Heading2:       convertTextBlock(block.Heading2),
		Heading3:       convertTextBlock(block.Heading3),
		Heading4:       convertTextBlock(block.Heading4),
		Heading5:       convertTextBlock(block.Heading5),
		Heading6:       convertTextBlock(block.Heading6),
		Heading7:       convertTextBlock(block.Heading7),
		Heading8:       convertTextBlock(block.Heading8),
		Heading9:       convertTextBlock(block.Heading9),
		Bullet:         convertTextBlock(block.Bullet),
		Ordered:        convertTextBlock(block.Ordered),
		Code:           convertTextBlock(block.Code),
		Quote:          convertTextBlock(block.Quote),
		Equation:       convertTextBlock(block.Equation),
		Todo:           convertTextBlock(block.Todo),
		Callout:        convertCallout(block.Callout),
		Image:          convertImage(block.Image),
		Table:          convertTable(block.Table),
		TableCell:      convertTableCell(block.TableCell),
		QuoteContainer: nilIfNil(block.QuoteContainer),
		Grid:           nilIfNil(block.Grid),
		GridColumn:     nilIfNil(block.GridColumn),
		Board:          convertBoard(block.Board),
	}
}

func convertTextBlock(text *larkdocx.Text) *TextBlock {
	if text == nil {
		return nil
	}
	elements := make([]*TextElement, 0, len(text.Elements))
	for _, element := range text.Elements {
		elements = append(elements, convertTextElement(element))
	}
	return &TextBlock{
		Style:    convertTextStyle(text.Style),
		Elements: elements,
	}
}

func convertTextStyle(style *larkdocx.TextStyle) *TextStyle {
	if style == nil {
		return nil
	}
	return &TextStyle{
		Align:            intValueOf(style.Align),
		Done:             boolValueOf(style.Done),
		Folded:           boolValueOf(style.Folded),
		Language:         intValueOf(style.Language),
		Wrap:             boolValueOf(style.Wrap),
		BackgroundColor:  valueOf(style.BackgroundColor),
		IndentationLevel: valueOf(style.IndentationLevel),
		Sequence:         valueOf(style.Sequence),
	}
}

func convertTextElement(element *larkdocx.TextElement) *TextElement {
	if element == nil {
		return nil
	}
	return &TextElement{
		TextRun:     convertTextRun(element.TextRun),
		MentionUser: convertMentionUser(element.MentionUser),
		MentionDoc:  convertMentionDoc(element.MentionDoc),
		Equation:    convertEquation(element.Equation),
	}
}

func convertTextRun(run *larkdocx.TextRun) *TextRun {
	if run == nil {
		return nil
	}
	return &TextRun{
		Content:          valueOf(run.Content),
		TextElementStyle: convertTextElementStyle(run.TextElementStyle),
	}
}

func convertTextElementStyle(style *larkdocx.TextElementStyle) *TextElementStyle {
	if style == nil {
		return nil
	}
	return &TextElementStyle{
		Bold:          boolValueOf(style.Bold),
		Italic:        boolValueOf(style.Italic),
		Strikethrough: boolValueOf(style.Strikethrough),
		Underline:     boolValueOf(style.Underline),
		InlineCode:    boolValueOf(style.InlineCode),
		Link:          convertLink(style.Link),
	}
}

func convertLink(link *larkdocx.Link) *Link {
	if link == nil {
		return nil
	}
	return &Link{URL: valueOf(link.Url)}
}

func convertMentionUser(user *larkdocx.MentionUser) *MentionUser {
	if user == nil {
		return nil
	}
	return &MentionUser{UserID: valueOf(user.UserId)}
}

func convertMentionDoc(doc *larkdocx.MentionDoc) *MentionDoc {
	if doc == nil {
		return nil
	}
	return &MentionDoc{
		Title: valueOf(doc.Title),
		URL:   valueOf(doc.Url),
	}
}

func convertEquation(equation *larkdocx.Equation) *EquationRef {
	if equation == nil {
		return nil
	}
	return &EquationRef{Content: valueOf(equation.Content)}
}

func convertCallout(callout *larkdocx.Callout) *CalloutBlock {
	if callout == nil {
		return nil
	}
	return &CalloutBlock{EmojiID: valueOf(callout.EmojiId)}
}

func convertImage(image *larkdocx.Image) *ImageBlock {
	if image == nil {
		return nil
	}
	return &ImageBlock{Token: valueOf(image.Token)}
}

func convertBoard(board *larkdocx.Board) *WhiteboardRef {
	if board == nil {
		return nil
	}
	return &WhiteboardRef{Token: valueOf(board.Token)}
}

func convertTable(table *larkdocx.Table) *TableBlock {
	if table == nil {
		return nil
	}
	property := &TableProperty{}
	if table.Property != nil {
		property.RowSize = intValueOf(table.Property.RowSize)
		property.ColumnSize = intValueOf(table.Property.ColumnSize)
		property.MergeInfo = make([]*TableMergeInfo, 0, len(table.Property.MergeInfo))
		for _, item := range table.Property.MergeInfo {
			property.MergeInfo = append(property.MergeInfo, convertTableMergeInfo(item))
		}
	}
	return &TableBlock{
		Cells:    table.Cells,
		Property: property,
	}
}

func convertTableMergeInfo(info *larkdocx.TableMergeInfo) *TableMergeInfo {
	if info == nil {
		return nil
	}
	return &TableMergeInfo{
		RowSpan: intValueOf(info.RowSpan),
		ColSpan: intValueOf(info.ColSpan),
	}
}

func convertTableCell(cell *larkdocx.TableCell) *TableCell {
	if cell == nil {
		return nil
	}
	return &TableCell{}
}

func convertDriveFiles(files []*larkdrive.File) []*DriveFile {
	result := make([]*DriveFile, 0, len(files))
	for _, file := range files {
		result = append(result, &DriveFile{
			Token: valueOf(file.Token),
			Name:  valueOf(file.Name),
			Type:  valueOf(file.Type),
			URL:   valueOf(file.Url),
		})
	}
	return result
}

func convertWikiNode(node *larkwiki.Node) *WikiNode {
	if node == nil {
		return nil
	}
	return &WikiNode{
		NodeToken: valueOf(node.NodeToken),
		ObjToken:  valueOf(node.ObjToken),
		ObjType:   valueOf(node.ObjType),
		Title:     valueOf(node.Title),
		HasChild:  boolValueOf(node.HasChild),
	}
}

func convertWikiNodes(nodes []*larkwiki.Node) []*WikiNode {
	result := make([]*WikiNode, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, convertWikiNode(node))
	}
	return result
}

func boolValueOf(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func intValueOf(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func valueOf(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nilIfNil[T any](value *T) *struct{} {
	if value == nil {
		return nil
	}
	return &struct{}{}
}

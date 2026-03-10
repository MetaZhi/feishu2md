package core

type AssetKind string

const (
	AssetKindImage      AssetKind = "image"
	AssetKindWhiteboard AssetKind = "whiteboard"
)

const (
	BlockTypePage           = 1
	BlockTypeText           = 2
	BlockTypeHeading1       = 3
	BlockTypeHeading2       = 4
	BlockTypeHeading3       = 5
	BlockTypeHeading4       = 6
	BlockTypeHeading5       = 7
	BlockTypeHeading6       = 8
	BlockTypeHeading7       = 9
	BlockTypeHeading8       = 10
	BlockTypeHeading9       = 11
	BlockTypeBullet         = 12
	BlockTypeOrdered        = 13
	BlockTypeCode           = 14
	BlockTypeQuote          = 15
	BlockTypeEquation       = 16
	BlockTypeTodo           = 17
	BlockTypeCallout        = 19
	BlockTypeDivider        = 22
	BlockTypeGrid           = 24
	BlockTypeGridColumn     = 25
	BlockTypeImage          = 27
	BlockTypeTable          = 31
	BlockTypeTableCell      = 32
	BlockTypeQuoteContainer = 34
)

type AssetRef struct {
	Kind  AssetKind `json:"kind"`
	Token string    `json:"token"`
}

type Document struct {
	DocumentID string `json:"document_id"`
	RevisionID int    `json:"revision_id,omitempty"`
	Title      string `json:"title,omitempty"`
}

type Block struct {
	BlockID        string         `json:"block_id"`
	ParentID       string         `json:"parent_id,omitempty"`
	Children       []string       `json:"children,omitempty"`
	BlockType      int            `json:"block_type"`
	Page           *TextBlock     `json:"page,omitempty"`
	Text           *TextBlock     `json:"text,omitempty"`
	Heading1       *TextBlock     `json:"heading1,omitempty"`
	Heading2       *TextBlock     `json:"heading2,omitempty"`
	Heading3       *TextBlock     `json:"heading3,omitempty"`
	Heading4       *TextBlock     `json:"heading4,omitempty"`
	Heading5       *TextBlock     `json:"heading5,omitempty"`
	Heading6       *TextBlock     `json:"heading6,omitempty"`
	Heading7       *TextBlock     `json:"heading7,omitempty"`
	Heading8       *TextBlock     `json:"heading8,omitempty"`
	Heading9       *TextBlock     `json:"heading9,omitempty"`
	Bullet         *TextBlock     `json:"bullet,omitempty"`
	Ordered        *TextBlock     `json:"ordered,omitempty"`
	Code           *TextBlock     `json:"code,omitempty"`
	Quote          *TextBlock     `json:"quote,omitempty"`
	Equation       *TextBlock     `json:"equation,omitempty"`
	Todo           *TextBlock     `json:"todo,omitempty"`
	Callout        *CalloutBlock  `json:"callout,omitempty"`
	Image          *ImageBlock    `json:"image,omitempty"`
	Table          *TableBlock    `json:"table,omitempty"`
	TableCell      *TableCell     `json:"table_cell,omitempty"`
	QuoteContainer *struct{}      `json:"quote_container,omitempty"`
	Grid           *struct{}      `json:"grid,omitempty"`
	GridColumn     *struct{}      `json:"grid_column,omitempty"`
	Board          *WhiteboardRef `json:"board,omitempty"`
}

type TextBlock struct {
	Style    *TextStyle     `json:"style,omitempty"`
	Elements []*TextElement `json:"elements,omitempty"`
}

type TextStyle struct {
	Align            int    `json:"align,omitempty"`
	Done             bool   `json:"done,omitempty"`
	Folded           bool   `json:"folded,omitempty"`
	Language         int    `json:"language,omitempty"`
	Wrap             bool   `json:"wrap,omitempty"`
	BackgroundColor  string `json:"background_color,omitempty"`
	IndentationLevel string `json:"indentation_level,omitempty"`
	Sequence         string `json:"sequence,omitempty"`
}

type TextElement struct {
	TextRun     *TextRun     `json:"text_run,omitempty"`
	MentionUser *MentionUser `json:"mention_user,omitempty"`
	MentionDoc  *MentionDoc  `json:"mention_doc,omitempty"`
	Equation    *EquationRef `json:"equation,omitempty"`
}

type TextRun struct {
	Content          string            `json:"content,omitempty"`
	TextElementStyle *TextElementStyle `json:"text_element_style,omitempty"`
}

type TextElementStyle struct {
	Bold          bool  `json:"bold,omitempty"`
	Italic        bool  `json:"italic,omitempty"`
	Strikethrough bool  `json:"strikethrough,omitempty"`
	Underline     bool  `json:"underline,omitempty"`
	InlineCode    bool  `json:"inline_code,omitempty"`
	Link          *Link `json:"link,omitempty"`
}

type Link struct {
	URL string `json:"url,omitempty"`
}

type MentionUser struct {
	UserID string `json:"user_id,omitempty"`
}

type MentionDoc struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

type EquationRef struct {
	Content string `json:"content,omitempty"`
}

type CalloutBlock struct {
	EmojiID string `json:"emoji_id,omitempty"`
}

type ImageBlock struct {
	Token string `json:"token,omitempty"`
}

type WhiteboardRef struct {
	Token string `json:"token,omitempty"`
}

type TableBlock struct {
	Cells    []string       `json:"cells,omitempty"`
	Property *TableProperty `json:"property,omitempty"`
}

type TableProperty struct {
	RowSize    int               `json:"row_size,omitempty"`
	ColumnSize int               `json:"column_size,omitempty"`
	MergeInfo  []*TableMergeInfo `json:"merge_info,omitempty"`
}

type TableMergeInfo struct {
	RowSpan int `json:"row_span,omitempty"`
	ColSpan int `json:"col_span,omitempty"`
}

type TableCell struct{}

type DriveFile struct {
	Token string `json:"token,omitempty"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
	URL   string `json:"url,omitempty"`
}

type WikiNode struct {
	NodeToken string `json:"node_token,omitempty"`
	ObjToken  string `json:"obj_token,omitempty"`
	ObjType   string `json:"obj_type,omitempty"`
	Title     string `json:"title,omitempty"`
	HasChild  bool   `json:"has_child,omitempty"`
}

package models

// ParsedContent represents the structured content extracted from a file.
// This structure should mirror the JSON format you intend to store in Document.Content
type ParsedContent struct {
	Type    string        `json:"type"` // e.g., "doc"
	Content []ContentNode `json:"content"`
}

// ContentNode represents a block-level element (paragraph, heading, list)
// or an inline element (text).
type ContentNode struct {
	Type    string        `json:"type"`              // e.g., "paragraph", "heading", "bulletList", "text"
	Level   int           `json:"level,omitempty"`   // For headings (1-6)
	Content []ContentNode `json:"content,omitempty"` // For container nodes like paragraph, heading, listItem
	Text    string        `json:"text,omitempty"`    // For text nodes
	Marks   []Mark        `json:"marks,omitempty"`   // For text nodes (bold, italic)
	// Add other attributes as needed (e.g., Attributes map[string]string for links, images)
}

// Mark represents inline formatting applied to a text node.
type Mark struct {
	Type string `json:"type"` // e.g., "bold", "italic"
	// Add attributes if marks need them (e.g., for links: Attrs map[string]string)
}

// // Example Usage (how you might build this structure)
// func buildExampleContent() *ParsedContent {
// 	return &ParsedContent{
// 		Type: "doc",
// 		Content: []ContentNode{
// 			{Type: "heading", Level: 1, Content: []ContentNode{{Type: "text", Text: "Main Title"}}},
// 			{Type: "paragraph", Content: []ContentNode{
// 				{Type: "text", Text: "This is "},
// 				{Type: "text", Text: "bold", Marks: []Mark{{Type: "bold"}}},
// 				{Type: "text", Text: " text."},
// 			}},
// 			{Type: "bulletList", Content: []ContentNode{
// 				{Type: "listItem", Content: []ContentNode{ // List item itself contains blocks
// 					{Type: "paragraph", Content: []ContentNode{{Type: "text", Text: "Item 1"}}},
// 				}},
// 			}},
// 		},
// 	}
// }

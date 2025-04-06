package docx

import (
	"encoding/xml"
	// Import other necessary packages
)

// --- Simplified OOXML Structures for document.xml ---
// These need refinement based on testing and OOXML specs. Namespaces omitted for simplicity.

type WordDocumentXML struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main document"`
	Body    *BodyXML `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main body"`
}

type BodyXML struct {
	XMLName xml.Name     `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main body"`
	Items   []BodyChoice // Use XMLName trick or explicit struct members for p, tbl etc.
	// We use a wrapper struct `BodyChoice` to handle different element types like <w:p>, <w:tbl>
}

// BodyChoice wrapper to handle mixed content in <w:body>
type BodyChoice struct {
	Paragraph *ParagraphXML `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main p"`
	Table     *TableXML     `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main tbl"`
}

type ParagraphXML struct {
	XMLName xml.Name        `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main p"`
	PPr     *ParagraphProps `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main pPr"`
	Items   []ParaChoice    // Use XMLName trick or explicit struct members for r, hyperlink etc.
}

// ParaChoice wrapper for mixed content in <w:p>
type ParaChoice struct {
	Run       *RunXML       `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main r"`
	Hyperlink *HyperlinkXML `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main hyperlink"`
}

type RunXML struct {
	XMLName xml.Name    `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main r"`
	RPr     *RunProps   `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main rPr"`
	Items   []RunChoice // Use XMLName trick or explicit struct members for t, br etc.
}

// RunChoice wrapper for mixed content in <w:r>
type RunChoice struct {
	Text  *TextXML  `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main t"`
	Break *BreakXML `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main br"`
	// Add other elements like Drawing <w:drawing> later for images
}

type TextXML struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main t"`
	Space   string   `xml:"http://www.w3.org/XML/1998/namespace space,attr,omitempty"` // Note namespace for xml:space
	Value   string   `xml:",chardata"`
}

type ParagraphProps struct {
	XMLName xml.Name    `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main pPr"`
	PStyle  *StyleIdVal `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main pStyle"`
	NumPr   *NumProps   `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main numPr"`
	// Add <w:jc> for Justification etc. if needed
}

type RunProps struct {
	XMLName xml.Name     `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main rPr"`
	RStyle  *StyleIdVal  `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main rStyle"`
	Bold    *OnOffToggle `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main b"`
	Italics *OnOffToggle `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main i"`
	// Add <w:color>, <w:sz> etc. if needed
}

type StyleIdVal struct {
	Val string `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main val,attr"`
}

type OnOffToggle struct {
	// We don't strictly need Val for simple presence check, but useful if val="false" exists
	Val *string `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main val,attr"`
}

type NumProps struct {
	Ilvl *struct {
		Val string `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main val,attr"`
	} `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main ilvl"`
	NumId *struct {
		Val string `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main val,attr"`
	} `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main numId"`
}

// Placeholders - Define further if needed
type TableXML struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main tbl"`
}
type HyperlinkXML struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main hyperlink"`
}
type BreakXML struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main br"`
}

// --- Helper Functions ---

// getParagraphStyleManual extracts style ID from unmarshaled properties.
func getParagraphStyleManual(pPr *ParagraphProps) string {
	if pPr != nil && pPr.PStyle != nil {
		return pPr.PStyle.Val // Returns the Style ID (e.g., "Heading1")
	}
	// NOTE: More robust parsing would involve checking default styles / document defaults
	return "Normal"
}

// isRunBoldManual checks for bold tag in unmarshaled properties.
func isRunBoldManual(rPr *RunProps) bool {
	if rPr != nil && rPr.Bold != nil {
		// Tag exists. Bold unless val="false" or val="0".
		if rPr.Bold.Val == nil || (*rPr.Bold.Val != "false" && *rPr.Bold.Val != "0") {
			return true
		}
	}
	// TODO: Check styles (rPr.RStyle, potentially paragraph style) for inheritance.
	return false
}

// isRunItalicManual checks for italic tag in unmarshaled properties.
func isRunItalicManual(rPr *RunProps) bool {
	if rPr != nil && rPr.Italics != nil {
		// Tag exists. Italic unless val="false" or val="0".
		if rPr.Italics.Val == nil || (*rPr.Italics.Val != "false" && *rPr.Italics.Val != "0") {
			return true
		}
	}
	// TODO: Check styles for inheritance.
	return false
}

// TODO: Add helper for list detection based on NumProps if implementing lists.
// func isParagraphListItemManual(pPr *ParagraphProps) (isList bool, numId string, level string) { ... }

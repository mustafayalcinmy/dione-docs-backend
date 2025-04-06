package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"log"

	"github.com/dione-docs-backend/internal/models" // Adjust import path
)

// manualParser implements the parser interface using standard libs.
// var _ parser.Parser = (*manualParser)(nil) // If using the interface

type manualParser struct{}

// NewManualParser creates a new parser using standard libs.
func NewManualParser() *manualParser {
	return &manualParser{}
}

// Parse implements the manual parsing logic.
// It expects an io.ReaderAt and size to read the zip archive.
func (p *manualParser) Parse(reader io.ReaderAt, size int64) (*models.ParsedContent, error) {
	log.Println("Starting DOCX manual parsing...")

	// --- Step 1: Open the ZIP archive ---
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip archive: %w", err)
	}

	// --- Step 2: Find and Read word/document.xml ---
	var docFile *zip.File
	for _, file := range zipReader.File {
		// NOTE: Consider also checking for "word/document2.xml" or similar as fallback? Unlikely needed.
		if file.Name == "word/document.xml" {
			docFile = file
			break
		}
	}

	if docFile == nil {
		return nil, fmt.Errorf("word/document.xml not found in archive")
	}

	docXMLReader, err := docFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open word/document.xml: %w", err)
	}
	defer docXMLReader.Close()

	// --- Step 3: Unmarshal word/document.xml ---
	// Define Go structs matching the OOXML structure (see utils.go)
	var wordDoc WordDocumentXML
	decoder := xml.NewDecoder(docXMLReader)
	// TODO: Handle namespaces if they cause issues during decoding.
	// You might need a custom decoder or struct tags with namespaces.
	err = decoder.Decode(&wordDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal document.xml: %w", err)
	}

	// --- Log the Unmarshaled Structure ---
	log.Printf("--- Unmarshaled wordDoc Structure (Check Body and Items) ---")
	log.Printf("%#v", wordDoc)
	if wordDoc.Body == nil {
		log.Println("!!! wordDoc.Body is NIL after unmarshal")
	} else {
		log.Printf("wordDoc.Body found. Number of items: %d", len(wordDoc.Body.Items))
		if len(wordDoc.Body.Items) == 0 {
			log.Println("!!! wordDoc.Body.Items is EMPTY after unmarshal")
		}
	}
	log.Println("---------------------------------------------------------")

	// --- Step 4: Prepare the result structure ---
	parsedContent := &models.ParsedContent{
		Type:    "doc",
		Content: []models.ContentNode{},
	}

	if wordDoc.Body == nil {
		log.Println("Aborting parsing loop as wordDoc.Body is nil.")
		return parsedContent, nil // Return the empty parsedContent early
	}

	// --- Step 5: Traverse the Unmarshaled XML Structure ---
	log.Printf("Starting loop through %d body items...", len(wordDoc.Body.Items))
	for i, element := range wordDoc.Body.Items {
		log.Printf("Processing Body Item Index: %d", i)

		// --- Handle Paragraphs (<w:p>) ---
		if paraXML := element.Paragraph; paraXML != nil {
			log.Printf("  Item %d: Identified as Paragraph <w:p>", i)

			paraNode := models.ContentNode{}
			styleName := getParagraphStyleManual(paraXML.PPr) // Use helper
			log.Printf("    Raw Style ID: '%s'", styleName)
			// TODO: Implement robust style mapping (styles.xml lookup?)
			switch styleName {
			case "Heading1":
				paraNode.Type = "heading"
				paraNode.Level = 1
			case "Heading2":
				paraNode.Type = "heading"
				paraNode.Level = 2
			case "Heading3":
				paraNode.Type = "heading"
				paraNode.Level = 3
			// TODO: Add list detection logic based on NumPr in paraXML.PPr
			// isList, numId, level := isParagraphListItemManual(paraXML.PPr) // Example
			// if isList { paraNode.Type = "listItem"; ... } else ...
			default:
				paraNode.Type = "paragraph"
			}
			paraNode.Content = []models.ContentNode{}
			log.Printf("    Created paraNode with Type: %s (Level: %d)", paraNode.Type, paraNode.Level)

			runFound := false // Flag to check if any runs are processed
			// Iterate through runs <w:r> within the paragraph
			for j, item := range paraXML.Items { // Assuming Items holds runs <w:r> etc.
				if runXML := item.Run; runXML != nil {
					runFound = true
					log.Printf("    Item %d, Run Index %d: Identified as Run <w:r>", i, j)

					// Iterate through elements within the run (<w:t>, <w:br>, etc.)
					textProcessed := false // Flag to check if text was found in run
					for k, runItem := range runXML.Items {
						if textXML := runItem.Text; textXML != nil {
							text := textXML.Value
							// Handle xml:space="preserve"
							if textXML.Space == "preserve" {
								// Keep leading/trailing spaces, maybe replace internal spaces with nbsp?
								// Simple approach: just use the value as is for now.
								log.Printf("      Item %d, Run %d, Text %d: Found text with xml:space=preserve", i, j, k)
							}

							if text != "" || textXML.Space == "preserve" { // Process even if empty if space=preserve?
								textProcessed = true
								isBold := isRunBoldManual(runXML.RPr)
								isItalic := isRunItalicManual(runXML.RPr)
								log.Printf("      Item %d, Run %d, Text %d: Text='%s', Bold=%t, Italic=%t", i, j, k, text, isBold, isItalic)

								textNode := models.ContentNode{Type: "text", Text: text}
								marks := []models.Mark{}
								if isBold {
									marks = append(marks, models.Mark{Type: "bold"})
								}
								if isItalic {
									marks = append(marks, models.Mark{Type: "italic"})
								}
								// TODO: Add other formatting checks

								if len(marks) > 0 {
									textNode.Marks = marks
								}
								paraNode.Content = append(paraNode.Content, textNode)
								log.Printf("      Appended textNode. Current paraNode.Content length: %d", len(paraNode.Content))

							} else {
								log.Printf("      Item %d, Run %d, Text %d: Empty text element found.", i, j, k)
							}
						} else if runItem.Break != nil {
							log.Printf("      Item %d, Run %d, Break %d: Found break <w:br/> (Handling TBD)", i, j, k)
							// TODO: Handle breaks - maybe map to newline in text or specific node?
						} else {
							log.Printf("      Item %d, Run %d, Other %d: Found non-text/non-break run item (Skipping)", i, j, k)
						}
					} // End run item loop
					if !textProcessed {
						log.Printf("    Item %d, Run Index %d: No text elements (<w:t>) found within this run.", i, j)
					}
				} else if item.Hyperlink != nil {
					log.Printf("    Item %d, Hyperlink %d: Found hyperlink <w:hyperlink> (Handling TBD)", i, j)
					// TODO: Handle hyperlinks - need to get relationship ID and look up URL
				} else {
					log.Printf("    Item %d, Other %d: Found non-run/non-hyperlink paragraph item (Skipping)", i, j)
				}
			} // End para item (run/hyperlink) loop
			if !runFound {
				log.Printf("  Item %d: No runs (<w:r>) found within this paragraph.", i)
			}

			// Only add paragraph node if it actually contains something?
			if len(paraNode.Content) > 0 {
				log.Printf("  Appending paraNode (Content len: %d) to main content.", len(paraNode.Content))
				parsedContent.Content = append(parsedContent.Content, paraNode)
				log.Printf("  Main content length is now: %d", len(parsedContent.Content))
			} else {
				log.Printf("  Skipping empty paraNode (Style: %s).", styleName)
			}
			log.Println("--------------------") // Separator

		} else if element.Table != nil {
			log.Printf("  Item %d: Identified as Table <w:tbl> (Skipping)", i)
			// TODO: Add table parsing logic if needed
		} else {
			log.Printf("  Item %d: Identified as unknown body element (Skipping)", i)
		}
	} // End body element loop

	log.Println("Finished DOCX manual parsing attempt.")
	return parsedContent, nil
}

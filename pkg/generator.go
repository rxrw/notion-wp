package pkg

//
//import (
//	"bytes"
//	"fmt"
//	"github.com/rxrw/notion-wp/platforms"
//	"github.com/rxrw/notion-wp/utils"
//	"io"
//	"log"
//	"strings"
//
//	"github.com/jomei/notionapi"
//)
//
//func emphFormat(a *notionapi.Annotations) (s string) {
//	s = "%s"
//	if a == nil {
//		return
//	}
//
//	if a.Code {
//		return "`%s`"
//	}
//
//	switch {
//	case a.Bold && a.Italic:
//		s = "***%s***"
//	case a.Bold:
//		s = "**%s**"
//	case a.Italic:
//		s = "*%s*"
//	}
//
//	if a.Underline {
//		s = "__" + s + "__"
//	} else if a.Strikethrough {
//		s = "~~" + s + "~~"
//	}
//
//	// TODO: color
//
//	return s
//}
//
//func ConvertRich(t notionapi.RichText) string {
//	switch t.Type {
//	case notionapi.ObjectTypeText:
//		if t.Text.Link != nil {
//			return fmt.Sprintf(
//				emphFormat(t.Annotations),
//				fmt.Sprintf("[%s](%s)", t.Text.Content, t.Text.Link.Url),
//			)
//		}
//		return fmt.Sprintf(emphFormat(t.Annotations), t.Text.Content)
//	case notionapi.ObjectTypeList:
//	}
//	return ""
//}
//
//func ConvertRichText(t []notionapi.RichText) string {
//	buf := &bytes.Buffer{}
//	for _, word := range t {
//		buf.WriteString(ConvertRich(word))
//	}
//
//	return buf.String()
//}
//
//func Generate(page notionapi.Page, blocks []notionapi.Block, config BlogConfig) ([]byte, error) {
//	buffer := &bytes.Buffer{}
//	GenerateContent(buffer, blocks, config)
//
//	return buffer.Bytes(), nil
//}
//
//func GenerateContent(w io.Writer, blocks []notionapi.Block, config BlogConfig, prefixes ...string) {
//	if len(blocks) == 0 {
//		return
//	}
//
//	numberedList := false
//	bulletedList := false
//
//	for _, block := range blocks {
//		// Add line break after list is finished
//		if bulletedList && block.GetType() != notionapi.BlockTypeBulletedListItem {
//			bulletedList = false
//			fmt.Fprintln(w)
//		}
//		if numberedList && block.GetType() != notionapi.BlockTypeNumberedListItem {
//			numberedList = false
//			fmt.Fprintln(w)
//		}
//
//		switch b := block.(type) {
//		case *notionapi.ParagraphBlock:
//			fprintln(w, prefixes, ConvertRichText(b.Paragraph.RichText)+"\n")
//			GenerateContent(w, b.Paragraph.Children, config)
//		case *notionapi.Heading1Block:
//			fprintf(w, prefixes, "# %s", ConvertRichText(b.Heading1.RichText))
//		case *notionapi.Heading2Block:
//			fprintf(w, prefixes, "## %s", ConvertRichText(b.Heading2.RichText))
//		case *notionapi.Heading3Block:
//			fprintf(w, prefixes, "### %s", ConvertRichText(b.Heading3.RichText))
//		case *notionapi.CalloutBlock:
//			if b.Callout.Icon != nil {
//				if b.Callout.Icon.Emoji != nil {
//					fprintf(w, prefixes, `{{%% callout emoji="%s" %%}}`, *b.Callout.Icon.Emoji)
//				} else {
//					fprintf(w, prefixes, `{{%% callout image="%s" %%}}`, b.Callout.Icon.GetURL())
//				}
//			}
//			fprintln(w, prefixes, ConvertRichText(b.Callout.RichText))
//			GenerateContent(w, b.Callout.Children, config, prefixes...)
//			fprintln(w, prefixes, "{{% /callout %}}")
//
//		case *notionapi.BookmarkBlock:
//			// Parse external page metadata
//			og, err := parseMetadata(b.Bookmark.URL)
//			if err != nil {
//				log.Println("error getting bookmark metadata:", err)
//			}
//
//			// GenerateContent shortcode with given metadata
//			fprintf(w, prefixes,
//				`{{< bookmark url="%s" title="%s" img="%s" >}}%s{{< /bookmark >}}`,
//				og.URL,
//				og.Title,
//				og.Image,
//				og.Description,
//			)
//
//		case *notionapi.QuoteBlock:
//			fprintf(w, prefixes, "> %s", ConvertRichText(b.Quote.RichText))
//			GenerateContent(w, b.Quote.Children, config,
//				append([]string{"> "}, prefixes...)...)
//			fprintln(w, prefixes)
//
//		case *notionapi.BulletedListItemBlock:
//			bulletedList = true
//			fprintf(w, prefixes, "- %s", ConvertRichText(b.BulletedListItem.RichText))
//			GenerateContent(w, b.BulletedListItem.Children, config,
//				append([]string{"    "}, prefixes...)...)
//
//		case *notionapi.NumberedListItemBlock:
//			numberedList = true
//			fprintf(w, prefixes, "1. %s", ConvertRichText(b.NumberedListItem.RichText))
//			GenerateContent(w, b.NumberedListItem.Children, config,
//				append([]string{"    "}, prefixes...)...)
//
//		case *notionapi.ImageBlock:
//			image, ct, fn, _ := utils.GetMedia(b.Image.GetURL())
//			wordpressClient, _ := platforms.NewWordpressUtil(config.WordPressConfig.Username, config.WordPressConfig.Password, config.WordPressConfig.SiteURL, nil)
//			media := wordpressClient.UploadMedia(fn, image, ct)
//			fprintf(w, prefixes, "![%s](%s)\n", ConvertRichText(b.Image.Caption), media.Link)
//
//		case *notionapi.CodeBlock:
//			if b.Code.Language == "plain text" {
//				fprintln(w, prefixes, "```")
//			} else {
//				fprintf(w, prefixes, "```%s", b.Code.Language)
//			}
//			fprintln(w, prefixes, ConvertRichText(b.Code.RichText))
//			fprintln(w, prefixes, "```")
//
//		case *notionapi.TableBlock:
//			rows := b.Table.Children
//			if len(rows) == 0 {
//				continue
//			}
//			headerLine := false
//			for index, row1 := range rows {
//				row := row1.(*notionapi.TableRowBlock)
//				if index == 1 && !headerLine {
//					fprintf(w, prefixes, "|%s", strings.Repeat("---|", len(row.TableRow.Cells)))
//					headerLine = true
//				} else {
//					cells := row.TableRow.Cells
//					line := ""
//					for _, cell := range cells {
//						line += fmt.Sprintf("|%s", ConvertRichText(cell))
//					}
//					line += "|"
//					fprintln(w, prefixes, line)
//				}
//			}
//			fprintln(w, prefixes)
//		case *notionapi.UnsupportedBlock:
//			if b.GetType() != "unsupported" {
//				fmt.Println("ℹ Unimplemented block", b.GetType())
//			} else {
//				fmt.Println("ℹ Unsupported block type")
//			}
//		default:
//			fmt.Println("ℹ Unimplemented block", b.GetType())
//		}
//
//	}
//}

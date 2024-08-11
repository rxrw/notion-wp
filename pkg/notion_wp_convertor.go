package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jomei/notionapi"
	"github.com/rxrw/notion-wp/platforms"
	"github.com/rxrw/notion-wp/utils"
)

type WordPressBlock struct {
	BlockName    string                 `json:"blockName"`
	Attrs        map[string]interface{} `json:"attrs,omitempty"`
	InnerBlocks  []WordPressBlock       `json:"innerBlocks,omitempty"`
	InnerHTML    string                 `json:"innerHTML"`
	InnerContent []string               `json:"innerContent"`
}

func Generate(page notionapi.Page, blocks []notionapi.Block, config BlogConfig) ([]byte, error) {
	wpBlocks := GenerateContent(blocks, config)

	content := ""
	for _, block := range wpBlocks {
		blockAttrs, _ := json.Marshal(block.Attrs)
		if string(blockAttrs) == "null" {
			content += fmt.Sprintf("<!-- wp:%s -->\n%s\n<!-- /wp:%s -->\n",
				block.BlockName, block.InnerHTML, block.BlockName)
		} else {
			content += fmt.Sprintf("<!-- wp:%s %s -->\n%s\n<!-- /wp:%s -->\n",
				block.BlockName, blockAttrs, block.InnerHTML, block.BlockName)
		}
	}

	post := fmt.Sprintf("<!-- wp:group -->\n<div class=\"wp-block-group\">%s</div>\n<!-- /wp:group -->",
		content)

	return []byte(post), nil
}

func GenerateContent(blocks []notionapi.Block, config BlogConfig) []WordPressBlock {
	var wpBlocks []WordPressBlock

	for _, block := range blocks {
		wpBlock := convertBlock(block, config)
		if wpBlock != nil {
			wpBlocks = append(wpBlocks, *wpBlock)
		}
	}

	return wpBlocks
}

func emphFormat(a *notionapi.Annotations) (s string) {
	s = "%s"
	if a == nil {
		return
	}

	if a.Code {
		return "`%s`"
	}

	switch {
	case a.Bold && a.Italic:
		s = "***%s***"
	case a.Bold:
		s = "**%s**"
	case a.Italic:
		s = "*%s*"
	}

	if a.Underline {
		s = "__" + s + "__"
	} else if a.Strikethrough {
		s = "~~" + s + "~~"
	}

	// TODO: color

	return s
}

func ConvertRich(t notionapi.RichText) string {
	switch t.Type {
	case notionapi.ObjectTypeText:
		if t.Text.Link != nil {
			return fmt.Sprintf(
				emphFormat(t.Annotations),
				fmt.Sprintf("[%s](%s)", t.Text.Content, t.Text.Link.Url),
			)
		}
		return fmt.Sprintf(emphFormat(t.Annotations), t.Text.Content)
	case notionapi.ObjectTypeList:
	}
	return ""
}

func ConvertRichText(t []notionapi.RichText) string {
	buf := &bytes.Buffer{}
	for _, word := range t {
		buf.WriteString(ConvertRich(word))
	}

	return buf.String()
}

func convertBlock(block notionapi.Block, config BlogConfig) *WordPressBlock {
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return &WordPressBlock{
			BlockName:    "core/paragraph",
			InnerHTML:    fmt.Sprintf("<p>%s</p>", ConvertRichText(b.Paragraph.RichText)),
			InnerContent: []string{fmt.Sprintf("<p>%s</p>", ConvertRichText(b.Paragraph.RichText))},
		}
	case *notionapi.Heading1Block:
		return &WordPressBlock{
			BlockName:    "core/heading",
			Attrs:        map[string]interface{}{"level": 2},
			InnerHTML:    fmt.Sprintf("<h2>%s</h2>", ConvertRichText(b.Heading1.RichText)),
			InnerContent: []string{fmt.Sprintf("<h2>%s</h2>", ConvertRichText(b.Heading1.RichText))},
		}
	case *notionapi.Heading2Block:
		return &WordPressBlock{
			BlockName:    "core/heading",
			Attrs:        map[string]interface{}{"level": 3},
			InnerHTML:    fmt.Sprintf("<h3>%s</h3>", ConvertRichText(b.Heading2.RichText)),
			InnerContent: []string{fmt.Sprintf("<h3>%s</h3>", ConvertRichText(b.Heading2.RichText))},
		}
	case *notionapi.Heading3Block:
		return &WordPressBlock{
			BlockName:    "core/heading",
			Attrs:        map[string]interface{}{"level": 4},
			InnerHTML:    fmt.Sprintf("<h4>%s</h4>", ConvertRichText(b.Heading3.RichText)),
			InnerContent: []string{fmt.Sprintf("<h4>%s</h4>", ConvertRichText(b.Heading3.RichText))},
		}
	case *notionapi.ImageBlock:
		image, ct, fn, _ := utils.GetMedia(b.Image.GetURL())
		wordpressClient, _ := platforms.NewWordpressUtil(config.WordPressConfig.Username, config.WordPressConfig.Password, config.WordPressConfig.SiteURL, nil)
		media := wordpressClient.UploadMedia(fn, image, ct)
		return &WordPressBlock{
			BlockName:    "core/image",
			Attrs:        map[string]interface{}{"url": media.Link},
			InnerHTML:    fmt.Sprintf("<figure class=\"wp-block-image\"><img src=\"%s\" alt=\"%s\"/></figure>", media.Link, ConvertRichText(b.Image.Caption)),
			InnerContent: []string{fmt.Sprintf("<figure class=\"wp-block-image\"><img src=\"%s\" alt=\"%s\"/></figure>", media.Link, ConvertRichText(b.Image.Caption))},
		}
	case *notionapi.BulletedListItemBlock:
		return &WordPressBlock{
			BlockName: "core/list-item",

			InnerHTML:    fmt.Sprintf("<li>%s</li>", ConvertRichText(b.BulletedListItem.RichText)),
			InnerContent: []string{fmt.Sprintf("<li>%s</li>", ConvertRichText(b.BulletedListItem.RichText))},
		}
	case *notionapi.NumberedListItemBlock:
		return &WordPressBlock{
			BlockName:    "core/list-item",
			InnerHTML:    fmt.Sprintf("<li>%s</li>", ConvertRichText(b.NumberedListItem.RichText)),
			InnerContent: []string{fmt.Sprintf("<li>%s</li>", ConvertRichText(b.NumberedListItem.RichText))},
		}
	case *notionapi.QuoteBlock:
		return &WordPressBlock{
			BlockName:    "core/quote",
			InnerHTML:    fmt.Sprintf("<blockquote class=\"wp-block-quote\"><p>%s</p></blockquote>", ConvertRichText(b.Quote.RichText)),
			InnerContent: []string{fmt.Sprintf("<blockquote class=\"wp-block-quote\"><p>%s</p></blockquote>", ConvertRichText(b.Quote.RichText))},
		}
	case *notionapi.CodeBlock:
		return &WordPressBlock{
			BlockName:    "core/code",
			Attrs:        map[string]interface{}{"language": b.Code.Language},
			InnerHTML:    fmt.Sprintf("<pre class=\"wp-block-code\"><code>%s</code></pre>", ConvertRichText(b.Code.RichText)),
			InnerContent: []string{fmt.Sprintf("<pre class=\"wp-block-code\"><code>%s</code></pre>", ConvertRichText(b.Code.RichText))},
		}
	// 可以继续添加更多区块类型的转换...
	default:
		fmt.Printf("Unsupported block type: %s\n", block.GetType())
		return nil
	}
}

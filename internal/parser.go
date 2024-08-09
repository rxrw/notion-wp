package internal

import (
	"context"
	"fmt"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/rxrw/notion-wp/pkg"
	"github.com/rxrw/notion-wp/platforms"
	"log"
	"os"

	"github.com/janeczku/go-spinner"
	"github.com/jomei/notionapi"
)

// func filterFromConfig(config notion_blog.BlogConfig) *notionapi.OrCompoundFilter {
// 	if config.FilterProp == "" || len(config.FilterValue) == 0 {
// 		return nil
// 	}

// 	properties := make(notionapi.OrCompoundFilter, len(config.FilterValue))

// 	for i, val := range config.FilterValue {
// 		properties[i] = notionapi.PropertyFilter{
// 			Property: config.FilterProp,
// 			Select: &notionapi.SelectFilterCondition{
// 				Equals: val,
// 			},
// 		}
// 	}

// 	return &properties
// }

func recursiveGetChildren(client *notionapi.Client, blockID notionapi.BlockID) (blocks []notionapi.Block, err error) {
	res, err := client.Block.GetChildren(context.Background(), blockID, &notionapi.Pagination{
		PageSize: 100,
	})
	if err != nil {
		return nil, err
	}

	blocks = res.Results
	if len(blocks) == 0 {
		return
	}

	for _, block := range blocks {
		switch b := block.(type) {
		case *notionapi.ParagraphBlock:
			b.Paragraph.Children, err = recursiveGetChildren(client, b.ID)
		case *notionapi.CalloutBlock:
			b.Callout.Children, err = recursiveGetChildren(client, b.ID)
		case *notionapi.QuoteBlock:
			b.Quote.Children, err = recursiveGetChildren(client, b.ID)
		case *notionapi.BulletedListItemBlock:
			b.BulletedListItem.Children, err = recursiveGetChildren(client, b.ID)
		case *notionapi.NumberedListItemBlock:
			b.NumberedListItem.Children, err = recursiveGetChildren(client, b.ID)
		case *notionapi.TableBlock:
			b.Table.Children, err = recursiveGetChildren(client, b.ID)
		}

		if err != nil {
			return
		}
	}

	return
}

func ParseAndGenerate(config pkg.BlogConfig) error {
	client := notionapi.NewClient(notionapi.Token(os.Getenv("NOTION_SECRET")))

	wordpressClient, _ := platforms.NewWordpressUtil(config.WordPressConfig.Username, config.WordPressConfig.Password, config.WordPressConfig.SiteURL, client)

	spin := spinner.StartNew("Querying Notion database")
	q, err := client.Database.Query(context.Background(), notionapi.DatabaseID(config.DatabaseID),
		&notionapi.DatabaseQueryRequest{
			// Filter:   filterFromConfig(config),
			PageSize: 100,
		})
	spin.Stop()
	if err != nil {
		return fmt.Errorf("❌ Querying Notion database: %s", err)
	}
	fmt.Println("✔ Querying Notion database: Completed")

	for i, res := range q.Results {
		title := pkg.ConvertRichText(res.Properties["Name"].(*notionapi.TitleProperty).Title)
		// platformOptions := res.Properties["Platform"].(*notionapi.MultiSelectProperty).MultiSelect
		// var platforms []string
		// for _, option := range platformOptions {
		// 	platforms = append(platforms, option.Name)
		// }

		fmt.Printf("-- Article [%d/%d] --\n", i+1, len(q.Results))
		spin = spinner.StartNew("Getting blocks tree")
		// Get page blocks tree
		blocks, err := recursiveGetChildren(client, notionapi.BlockID(res.ID))
		spin.Stop()
		if err != nil {
			log.Println("❌ Getting blocks tree:", err)
			continue
		}
		fmt.Println("✔ Getting blocks tree: Completed")

		markdownContent, err := pkg.Generate(res, blocks, config)
		if err != nil {
			fmt.Println("Generating Failed")
		}
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		p := parser.NewWithExtensions(extensions)

		htmlFlags := html.CommonFlags | html.HrefTargetBlank
		opts := html.RendererOptions{Flags: htmlFlags}
		renderer := html.NewRenderer(opts)
		resource := markdown.ToHTML(markdownContent, p, renderer)

		var tags []string

		for _, tag := range res.Properties["Tags"].(*notionapi.MultiSelectProperty).MultiSelect {
			tags = append(tags, tag.Name)
		}

		// Upload to WordPress
		var imageURL string
		if res.Cover != nil {
			imageURL = res.Cover.GetURL()
		}
		fmt.Println("Start uploading: ", title)
		wordpressClient.UpdateOrCreatePost(title, resource, res.CreatedTime,
			[]string{res.Properties["Category"].(*notionapi.SelectProperty).Select.Name},
			tags,
			imageURL,
			res.Properties["Status"].(*notionapi.StatusProperty).Status.Name,
			int(res.Properties["WordPress ID"].(*notionapi.NumberProperty).Number),
			res.LastEditedTime,
		)
		fmt.Println("✔ Process completed: ", title)
	}

	return nil
}
func updateNotionPageWordPressID(client *notionapi.Client, p notionapi.Page, wordPressID int, wordpressLink string) bool {
	updatedProps := make(notionapi.Properties)
	updatedProps["WordPress ID"] = notionapi.NumberProperty{
		Number: float64(wordPressID),
	}
	updatedProps["WordPress Link"] = notionapi.URLProperty{
		URL: wordpressLink,
	}

	_, err := client.Page.Update(context.Background(), notionapi.PageID(p.ID),
		&notionapi.PageUpdateRequest{
			Properties: updatedProps,
		},
	)
	return err == nil
}

package internal

import (
	"context"
	"fmt"
	"github.com/rxrw/notion-wp/pkg"
	"github.com/rxrw/notion-wp/platforms"
	"log"
	"os"

	"github.com/janeczku/go-spinner"
	"github.com/jomei/notionapi"
)

func filterFromConfig(config pkg.BlogConfig) *notionapi.OrCompoundFilter {
	if config.FilterProp == "" || len(config.FilterValue) == 0 {
		return nil
	}

	properties := make(notionapi.OrCompoundFilter, len(config.FilterValue))

	for i, val := range config.FilterValue {
		properties[i] = notionapi.PropertyFilter{
			Property: config.FilterProp,
			MultiSelect: &notionapi.MultiSelectFilterCondition{
				Contains: val,
			},
		}
	}

	return &properties
}

func recursiveGetChildren(client *notionapi.Client, blockID notionapi.BlockID) ([]notionapi.Block, error) {
	var allBlocks []notionapi.Block
	var startCursor notionapi.Cursor

	for {
		// 获取当前页的块
		res, err := client.Block.GetChildren(context.Background(), blockID, &notionapi.Pagination{
			PageSize:    100,
			StartCursor: startCursor, // 设置分页的起始光标
		})
		if err != nil {
			return nil, err
		}

		// 将获取的块添加到全部块的列表中
		allBlocks = append(allBlocks, res.Results...)

		// 如果没有更多的块，则退出循环
		if res.HasMore {
			startCursor = notionapi.Cursor(res.NextCursor) // 更新光标为下一页的起点
		} else {
			break
		}
	}

	// 递归获取每个块的子块
	for _, block := range allBlocks {
		var err error
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
			return nil, err
		}
	}

	return allBlocks, nil
}

func ParseAndGenerate(config pkg.BlogConfig) error {
	client := notionapi.NewClient(notionapi.Token(os.Getenv("NOTION_SECRET")))

	wordpressClient, _ := platforms.NewWordpressUtil(config.WordPressConfig.Username, config.WordPressConfig.Password, config.WordPressConfig.SiteURL, client)

	var allResults []notionapi.Page
	var startCursor notionapi.Cursor

	for {
		spin := spinner.StartNew("Querying Notion database")
		q, err := client.Database.Query(context.Background(), notionapi.DatabaseID(config.DatabaseID),
			&notionapi.DatabaseQueryRequest{
				Filter:      filterFromConfig(config),
				PageSize:    100,
				StartCursor: startCursor, // 分页的起始光标
				Sorts: []notionapi.SortObject{
					{
						Property:  "Created",
						Timestamp: "created_time",
						Direction: "ascending",
					},
				},
			})
		spin.Stop()
		if err != nil {
			return fmt.Errorf("❌ Querying Notion database: %s", err)
		}
		fmt.Println("✔ Querying Notion database: Completed")

		// 将当前页的结果添加到总结果中
		allResults = append(allResults, q.Results...)

		// 如果有更多的结果，则更新光标继续请求
		if q.HasMore {
			startCursor = q.NextCursor
		} else {
			break
		}
	}

	for i, res := range allResults {
		if !wordpressClient.CheckIfShouldProcess(res) {
			continue
		}
		title := pkg.ConvertRichText(res.Properties["Name"].(*notionapi.TitleProperty).Title)
		// platformOptions := res.Properties["Platform"].(*notionapi.MultiSelectProperty).MultiSelect
		// var platforms []string
		// for _, option := range platformOptions {
		// 	platforms = append(platforms, option.Name)
		// }

		fmt.Printf("-- Article [%d/%d] --\n", i+1, len(allResults))
		spin := spinner.StartNew("Getting blocks tree")
		// Get page blocks tree
		blocks, err := recursiveGetChildren(client, notionapi.BlockID(res.ID))
		spin.Stop()
		if err != nil {
			log.Println("❌ Getting blocks tree:", err)
			continue
		}
		fmt.Println("✔ Getting blocks tree: Completed")

		wpRawContent, err := pkg.Generate(res, blocks, config)
		if err != nil {
			fmt.Println("Generating Failed")
		}

		var tags []string

		for _, tag := range res.Properties["Tags"].(*notionapi.MultiSelectProperty).MultiSelect {
			tags = append(tags, tag.Name)
		}

		var imageURL string
		if res.Cover != nil {
			imageURL = res.Cover.GetURL()
		}
		fmt.Println("Start uploading: ", title)
		wordpressClient.UpdateOrCreatePost(res, title, string(wpRawContent),
			[]string{res.Properties["Category"].(*notionapi.SelectProperty).Select.Name},
			tags,
			imageURL,
			res.Properties["Status"].(*notionapi.StatusProperty).Status.Name,
			int(res.Properties["WordPress ID"].(*notionapi.NumberProperty).Number),
		)
		fmt.Println("✔ Process completed: ", title)
	}

	return nil
}

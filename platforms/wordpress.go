package platforms

import (
	"context"
	"fmt"
	"github.com/jomei/notionapi"
	"github.com/rxrw/go-wordpress"
	"github.com/rxrw/notion-wp/utils"
	"strings"
)

var wordpressUtil *WordPressUtil

type WordPressUtil struct {
	wordpressApi     *wordpress.Client
	ctx              context.Context
	notionClient     *notionapi.Client
	cachedCategories []*wordpress.Category
	cachedTags       []*wordpress.Tag
}

func (w *WordPressUtil) CheckIfShouldProcess(notionPage notionapi.Page) bool {
	wordpressID := int(notionPage.Properties["WordPress ID"].(*notionapi.NumberProperty).Number)
	existingPost, _, _ := w.wordpressApi.Posts.Get(context.Background(), wordpressID, nil)
	if existingPost.Modified.Equal(notionPage.LastEditedTime) || existingPost.Modified.After(notionPage.LastEditedTime) {
		fmt.Printf("Post %s is already latest in WordPress, skipping\n", notionPage.Properties["Name"].(*notionapi.TitleProperty).Title[0].PlainText)
		return false
	}
	return true
}

func (w *WordPressUtil) UpdateOrCreatePost(notionPage notionapi.Page, title string, content string, categories []string, tags []string, bannerImageUrl string, status string, wordpressID int) int {
	var err error
	var post *wordpress.Post
	post = &wordpress.Post{
		Title:      wordpress.RenderedString{Raw: title},
		Content:    wordpress.RenderedString{Raw: content},
		Categories: w.GetCategoryIds(categories),
		Tags:       w.GetTagIds(tags),
		Status:     w.processWordpressStatus(status),
		Date:       wordpress.Time{Time: notionPage.CreatedTime},
	}

	if bannerImageUrl != "" {
		mediaData, contentType, filename, _ := utils.GetMedia(bannerImageUrl)
		media := w.UploadMedia(filename, mediaData, contentType)
		post.FeaturedMedia = media.ID
	}

	if wordpressID > 0 {
		fmt.Printf("Updating post %s in WordPress\n", title)
		post.ID = wordpressID
		post, _, err = w.wordpressApi.Posts.Update(context.Background(), wordpressID, post)
	} else {
		post.Date = wordpress.Time{Time: notionPage.CreatedTime}
		fmt.Printf("Creating post %s , %s in WordPress\n", title, notionPage.CreatedTime)
		post, _, err = w.wordpressApi.Posts.Create(context.Background(), post)
		w.updateNotionPageWordPressID(notionPage, post.ID)
	}
	if err != nil {
		fmt.Println(err)
	}
	return post.ID
}

func (w *WordPressUtil) UploadMedia(filename string, data []byte, contentType string) *wordpress.Media {
	media, _, err := w.wordpressApi.Media.Create(context.Background(), &wordpress.MediaUploadOptions{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		fmt.Println(err)
	}
	return media
}

func (w *WordPressUtil) GetCategoryIds(categories []string) []int {
	var categoryIds []int
	if w.cachedCategories == nil {
		w.cachedCategories, _, _ = w.wordpressApi.Categories.List(context.Background(), nil)
	}
	// 使用 cachedCategories 进行处理
	for _, category := range categories {
		found := false
		for _, wpCategory := range w.cachedCategories {
			if wpCategory.Name == category {
				categoryIds = append(categoryIds, wpCategory.ID)
				found = true
				break
			}
		}
		if !found {
			newCategory, _, _ := w.wordpressApi.Categories.Create(context.Background(), &wordpress.Category{
				Name: category,
			})
			// 添加新创建的分类到缓存
			w.cachedCategories = append(w.cachedCategories, newCategory)
			categoryIds = append(categoryIds, newCategory.ID)
		}
	}

	return categoryIds
}

func (w *WordPressUtil) GetTagIds(tags []string) []int {
	var tagIds []int
	if w.cachedTags == nil {
		w.cachedTags, _, _ = w.wordpressApi.Tags.List(context.Background(), nil)
	}
	// 使用 cachedTags 进行处理
	for _, tag := range tags {
		found := false
		for _, wpTag := range w.cachedTags {
			if wpTag.Name == tag {
				tagIds = append(tagIds, wpTag.ID)
				found = true
				break
			}
		}

		if !found {
			newTag, _, _ := w.wordpressApi.Tags.Create(context.Background(), &wordpress.Tag{
				Name: tag,
			})
			// 添加新创建的分类到缓存
			w.cachedTags = append(w.cachedTags, newTag)
			tagIds = append(tagIds, newTag.ID)
		}
	}

	return tagIds
}

func (w *WordPressUtil) updateNotionPageWordPressID(p notionapi.Page, wordpressID int) bool {
	updatedProps := make(notionapi.Properties)
	updatedProps["WordPress ID"] = notionapi.NumberProperty{
		Number: float64(wordpressID),
	}

	_, err := w.notionClient.Page.Update(context.Background(), notionapi.PageID(p.ID),
		&notionapi.PageUpdateRequest{
			Properties: updatedProps,
		},
	)
	return err == nil
}

func (w *WordPressUtil) processWordpressStatus(statusText string) string {
	if strings.Contains(statusText, "Draft") || strings.Contains(statusText, "In progress") {
		return wordpress.PostStatusDraft
	}
	if strings.Contains(statusText, "Published") {
		return wordpress.PostStatusPublish
	}
	return wordpress.PostStatusDraft
}

func NewWordpressUtil(username string, password string, siteUrl string, notionClient *notionapi.Client) (*WordPressUtil, error) {
	if wordpressUtil != nil {
		return wordpressUtil, nil
	}

	tp := wordpress.BasicAuthTransport{
		Username: username,
		Password: password,
	}

	wordpressClient, err := wordpress.NewClient(siteUrl, tp.Client())
	if err != nil {
		return nil, err
	}
	wordpressUtil = &WordPressUtil{
		wordpressApi: wordpressClient,
		ctx:          context.Background(),
		notionClient: notionClient,
	}
	return wordpressUtil, err
}

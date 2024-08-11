package pkg

type WordPressClient struct {
	Username string
	Password string
	SiteURL  string
}

type BlogConfig struct {
	DatabaseID string `usage:"ID of the Notion database of the blog."`

	WordPressConfig WordPressClient

	PropertyDescription string `usage:"Description property name in Notion."`
	PropertyTags        string `usage:"Tags multi-select property name in Notion."`
	PropertyCategory    string `usage:"Category select property name in Notion."`

	FilterProp  string   `usage:"Property of the filter to apply to a select value of the articles."`
	FilterValue []string `usage:"Value of the filter to apply to the Notion articles database."`
}

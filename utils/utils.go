package utils

import (
	"fmt"
	"github.com/janeczku/go-spinner"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func GetMedia(imgURL string) (_ []byte, contentType string, filename string, err error) {
	// Split image url to get host and file name
	splittedURL, err := url.Parse(imgURL)
	if err != nil {
		return nil, "", "", fmt.Errorf("malformed url: %s", err)
	}

	// Get file name
	name := splittedURL.Path

	spin := spinner.StartNew(fmt.Sprintf("Getting image `%s`", name))
	defer func() {
		spin.Stop()
		if err != nil {
			fmt.Printf("❌ Getting image `%s`: %s", name, err)
		} else {
			fmt.Printf("✔ Getting image `%s`: Completed", name)
		}
	}()

	resp, err := http.Get(imgURL)
	if err != nil {
		return nil, "", "", fmt.Errorf("couldn't download image: %s", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	filename = strings.Split(name, "/")[len(strings.Split(name, "/"))-1]
	return data, resp.Header.Get("Content-Type"), name, nil
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ericaro/frontmatter"
	"github.com/metal3d/go-slugify"
	"github.com/mmcdole/gofeed"
)

type feed struct {
	Name string
	URL  string
}

type feedItem struct {
	Title      string
	Content    string `fm:"content" yaml:"-"`
	Date       string
	Author     string
	AuthorLink string `yaml:"authorlink"`
	Slug       string `yaml:"-"`
	Tags       []string
}

var (
	wg sync.WaitGroup

	// Show up to 10 days of posts
	relevantDuration = 10 * 24 * time.Hour

	sourceJSON = "../feeds.json"
	outputDir  = "../news-site/content/post" // So we can feed them to Hugo

	// Error out if fetching feeds takes longer than a minute
	timeout = time.Minute
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	byteValue, err := ioutil.ReadFile(sourceJSON)
	if err != nil {
		return err
	}

	var feeds []feed
	if err := json.Unmarshal(byteValue, &feeds); err != nil {
		return err
	}

	wg.Add(len(feeds))

	for _, feed := range feeds {
		go getFeedPosts(ctx, feed)
	}

	wg.Wait()

	return nil
}

// getFeedPosts handles all posts from a feed from the last `relevantDuration` time period.
func getFeedPosts(ctx context.Context, feed feed) {
	defer wg.Done()

	parser := gofeed.NewParser()
	feedRes, err := parser.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		log.Printf("[Error] Error parsing feed '%s': %s\n", feed.Name, err.Error())
		return
	}

	// feedDir := path.Join(outputDir, slugify.Marshal(feed.Name))
	feedDir := path.Join(outputDir)
	if err := os.MkdirAll(feedDir, 0700); err != nil {
		log.Printf("[Error] Error creating news folder for feed '%s': %s\n", feed.Name, err.Error())
		return
	}

	var postsCount = 0
	for _, item := range feedRes.Items {
		post, err := parseFeedItem(feed, item)
		if err != nil {
			break
		}

		if err := storePost(feedDir, post); err != nil {
			log.Printf("[Error] Error saving post: %s\n", err.Error())
			continue
		}

		postsCount++
	}

	log.Printf("Saved %d posts for feed '%s'. Source had %d.\n", postsCount, feed.Name, len(feedRes.Items))
}

func parseFeedItem(feed feed, item *gofeed.Item) (*feedItem, error) {
	published := item.PublishedParsed
	if published == nil {
		published = item.UpdatedParsed
	}
	if published.Before(time.Now().Add(-relevantDuration)) {
		return nil, errors.New("Skipped")
	}

	post := &feedItem{
		Title:      item.Title,
		Date:       published.Format("2006-01-02 15:04:05"),
		Slug:       slugify.Marshal(item.Title),
		Author:     feed.Name,
		AuthorLink: item.Link,
		Tags:       []string{slugify.Marshal(feed.Name)},
	}

	if len(item.Description) != 0 {
		post.Content = item.Description
	} else if len(item.Content) != 0 {
		post.Content = item.Content
	}

	return post, nil
}

func storePost(folder string, post *feedItem) error {
	data, err := frontmatter.Marshal(post)
	if err != nil {
		return err
	}

	fileName := strings.Join([]string{post.Slug, "md"}, ".")
	if err = ioutil.WriteFile(path.Join(folder, fileName), data, 0700); err != nil {
		return err
	}

	return nil
}

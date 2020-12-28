package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ericaro/frontmatter"
)

type feedItem struct {
	Content string `fm:"content" yaml:"-"`
	Date    string
}

var (
	// Show up to x days of posts
	relevantDuration = 5 * (24 * time.Hour)

	outputDir = "../news-site/content/post" // So we can feed them to Hugo

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
	totalFilesCnt := 0
	deletedFilesCnt := 0

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("[Error] Error opening posts folder: %s\n", err.Error())
			return err
		}

		if !info.IsDir() {
			totalFilesCnt++

			toDelete, err := checkPostFile(path)
			if err != nil {
				log.Printf("[Error] Error checking post at %s: %s\n", path, err.Error())
				return nil
			}

			if toDelete {
				if err := os.Remove(path); err != nil {
					log.Printf("[Error] Error removing file at %s: %s\n", path, err.Error())
					return nil
				}

				deletedFilesCnt++
			}
		}

		return nil
	})

	if err == nil {
		log.Printf("Total files: %d, Deleted files: %d", totalFilesCnt, deletedFilesCnt)
	}

	return err
}

func checkPostFile(filePath string) (bool, error) {
	log.Printf("Checking file: %s\n", filePath)

	byteValue, err := ioutil.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	item := new(feedItem)
	err = frontmatter.Unmarshal(byteValue, item)
	if err != nil {
		return false, err
	}

	postDate, err := time.Parse("2006-01-02 15:04:05", item.Date)
	if err != nil {
		return false, err
	}

	if postDate.Before(time.Now().Add(-relevantDuration)) {
		log.Printf("File at %s will be deleted", filePath)
		return true, nil
	}

	return false, nil
}

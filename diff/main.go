package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func main() {

	rootDir, _ := filepath.Abs(filepath.Join("./../"))

	soundcloudRssUrl := "http://feeds.soundcloud.com/users/soundcloud:users:656797185/sounds.rss"
	resp, err := http.Get(soundcloudRssUrl)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	indexFile, err := os.Open(filepath.Join(rootDir, "docs/index.xml"))
	inputData, err := io.ReadAll(indexFile)
	if err != nil {
		panic(err)
	}

	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain(string(data), string(inputData), false)
	diffPath := filepath.Join(rootDir, "docs/change.diff")
	os.Remove(diffPath)
	diffFile, _ := os.Create(diffPath)

	io.WriteString(diffFile, dmp.DiffPrettyText(diffs))

	fmt.Println("write to ", diffPath)
}

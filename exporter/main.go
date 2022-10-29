package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"
    "strings"
    "strconv"
)

type Rss struct {
	XMLName     xml.Name `xml:"rss"`
	Version     string   `xml:"version,attr"`
	Channel     Channel  `xml:"channel"`
	Description string   `xml:"description"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
}

type Channel struct {
	XMLName     xml.Name `xml:"channel"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Items       []Item   `xml:"item"`
}

type Item struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate"`
	Guid        string   `xml:"guid"`
	Duration    string   `xml:"duration"`
	Author      string   `xml:"author"`
	Explicit    string   `xml:"explicit"`
	Summary     string   `xml:"summary"`
	Subtitle    string   `xml:"subtitle"`
}

const (
	layoutHeader = `---
layout: episode
title: "{{ .Title }}"
date: {{ .Date }}
people:
  - volyx
  - evgenii_borisov
audio: 0-javaswag-evgenii-borisov.mp3
guid: {{ .Guid }}
image: images/logo.png
description: {{ .Description }}
---`
)

type Episode struct {
    Number      int
	Title       string
	Date        string
	Guid        string
	Description string
}

func main() {
	sourceFile, err := os.Open("./../layout/soundcloud_rss.xml")

	if err != nil {
		fmt.Println(err)
	}

	rss := &Rss{}
	blob, err := io.ReadAll(sourceFile)
	if err := xml.Unmarshal([]byte(blob), &rss); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(rss.Channel.Items); i++ {
		item := rss.Channel.Items[i]
        fmt.Println(getNumber(item.Title))
		fmt.Println(item.Title)
		fmt.Println(item.Duration)
	}

	episodeDir := filepath.Dir("./../content/episode/")
	files := []string{}
	err = filepath.Walk(episodeDir, func(path string, info os.FileInfo, err error) error {
		fullpath, err := filepath.Abs(path)
		files = append(files, fullpath)
		return nil
	})

	for _, path := range files {
		fmt.Println(path)
	}

	episode := Episode{0, "John", "a regular user", "", ""}

	ut, err := template.New("episodes").Parse(layoutHeader)

	if err != nil {
		panic(err)
	}

    episodePath := filepath.Join(episodeDir, fmt.Sprintf("_%v.txt", episode.Number))

    os.Remove(episodePath)

    file, _ := os.Create(episodePath)

	err = ut.Execute(file, episode)

	if err != nil {
		panic(err)
	}
}

func getNumber(title string) (int, error) {
    // #Number - GuestName - EpisodeName
    parts := strings.Split(title, "-")
    numberPart := strings.Trim(parts[0], " ")
    number, err := strconv.Atoi(numberPart[1:])
    if err != nil {
        return 0, err
    }
    return number, nil
}

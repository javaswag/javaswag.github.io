package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type ListBucketResult struct {
	XMLName  xml.Name  `xml:"ListBucketResult"`
	Name     string    `xml:"Name"`
	Contents []Content `xml:"Contents"`
}

type Content struct {
	XMLName      xml.Name `xml:"Contents"`
	Key          string   `xml:"Key"`
	LastModified string   `xml:"LastModified"`
	ETag         string   `xml:"ETag"`
	Size         string   `xml:"Size"`
}

type Audio struct {
    Number int
    Name   string
}

type AudioList []Audio

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
  - {{ .Guest}}
audio: {{ .Audio}}
guid: {{ .Guid }}
image: images/logo.png
description: {{ .Description }}
---

## {{ .Title }}

{{ .Description }}`
)

const (
	dateFormat = "Mon, 02 Jan 2006 15:04:05 -0700"
)

type Episode struct {
	Number      int
	Title       string
	Date        string
	Guid        string
    Guest       string
    Audio       string
	Description string
}

type Episodes []Episode

func (d Episodes) Len() int {
	return len(d)
}

func (d Episodes) Less(i, j int) bool {
	return d[i].Number < d[j].Number
}

func (d Episodes) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d AudioList) Len() int {
    return len(d)
}

func (d AudioList) Less(i, j int) bool {
    return d[i].Number < d[j].Number
}

func (d AudioList) Swap(i, j int) {
    d[i], d[j] = d[j], d[i]
}


func main() {

    audioList := fetchAudioList()

	sourceFile, err := os.Open("./../layout/soundcloud_rss.xml")

	if err != nil {
		fmt.Println(err)
	}

	rss := &Rss{}
	blob, err := io.ReadAll(sourceFile)
	if err := xml.Unmarshal([]byte(blob), &rss); err != nil {
		log.Fatal(err)
	}

	episodes := []Episode{}

	for i := 0; i < len(rss.Channel.Items); i++ {
		item := rss.Channel.Items[i]
		number, _ := getNumber(item.Title)
		// Tue, 15 Feb 2022 17:18:17 +0000
		t, err := time.Parse(dateFormat, item.PubDate)
		if err != nil {
			fmt.Println("Error while parsing date :", err)
		}
        guest, _ := getGuestName(audioList[number].Name)
		episode := Episode{
			Number:      number,
			Title:       item.Title,
			Date:        t.Format("2006-01-02"),
			Guid:        item.Guid,
            Guest:       guest,
            Audio:       audioList[number].Name,   
			Description: item.Description,
		}
		episodes = append(episodes, episode)
	}

	sort.Sort(Episodes(episodes))

	episodeDir := filepath.Dir("./../content/episode/")

	for i := 0; i < len(episodes) && i < 5; i++ {
		episode := episodes[i]

		files := []string{}
		err = filepath.Walk(episodeDir, func(path string, info os.FileInfo, err error) error {
			fullpath, err := filepath.Abs(path)
			files = append(files, fullpath)
			return nil
		})

		ut, err := template.New("episodes").Parse(layoutHeader)

		if err != nil {
			panic(err)
		}

		episodePath := filepath.Join(episodeDir, fmt.Sprintf("_%v.md", episode.Number))

		os.Remove(episodePath)

		file, _ := os.Create(episodePath)

		err = ut.Execute(file, episode)

		if err != nil {
			panic(err)
		}
	}
}

func fetchAudioList() []Audio {
    resp, err := http.Get("https://storage.yandexcloud.net/javaswag/?list-type")

    if err != nil {
        panic(err)
    }

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        panic(err)
    }

    audioList := []Audio{}

    xmlBucket := &ListBucketResult{}
    if err := xml.Unmarshal([]byte(body), &xmlBucket); err != nil {
        panic(err)
    }
    for i := 0; i < len(xmlBucket.Contents); i++ {
        content := xmlBucket.Contents[i]
        if !strings.Contains(content.Key, "-") {
            continue
        }
        parts := strings.Split(content.Key, "-")
        if len(parts) == 0 {
            continue
        }
        audioNumber, _ := strconv.Atoi(parts[0])
        audio := Audio {
            Number: audioNumber,
            Name: content.Key,
        }
        audioList = append(audioList, audio)
    }

    sort.Sort(AudioList(audioList))
    return audioList
}

func getGuestName(audio string) (string, error) {
    parts := strings.Split(audio, ".")
    words := strings.Split(parts[0], "-")
    return fmt.Sprintf("%s-%s", words[len(words) - 2], words[len(words) - 1]), nil
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

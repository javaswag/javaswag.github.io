package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	Size   string
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
number: {{ .Number }}
title: "{{ .Title }}"
date: {{ .Date }}
people:
  - volyx
  - {{ .Guest}}
audio: {{ .Audio}}
size: {{ .Size }}
guid: {{ .Guid }}
image: images/logo.png
description: {{ .Description }}
duration: {{ .Duration }}
draft: false
---

{{ .Content  }}`
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
	Size        string
	Description string
	Duration    string
	Content     string
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

	rootDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	soundcloudRssUrl := "http://feeds.soundcloud.com/users/soundcloud:users:656797185/sounds.rss"
	episodeDir := filepath.Join(rootDir, "/content/episode/")
	audioS3Url := "https://storage.yandexcloud.net/javaswag/?list-type"

	_, isHugo := os.LookupEnv("USE_HUGO")

	fmt.Println("start from", rootDir, "hugo", isHugo)

	audioList := fetchAudioList(audioS3Url)

	os.Mkdir(filepath.Join(rootDir, "/docs"), 0750)

	rss := fetchSoundcloudRss(filepath.Join(rootDir, "/docs", "soundcloud.xml"), soundcloudRssUrl)

	episodes := []Episode{}

	for i := 0; i < len(rss.Channel.Items); i++ {
		item := rss.Channel.Items[i]
		number, err := getNumber(item.Title)
		if err != nil {
			panic(err)
		}
		var t time.Time
		t, err = time.Parse(dateFormat, item.PubDate)
		if err != nil {
			panic(err)
		}
		if len(audioList) <= number {
			panic(fmt.Sprintf("could not find audio for index %d", number))
		}
		var guest string
		guest, err = getGuestName(audioList[number].Name)
		if err != nil {
			panic(err)
		}
		htmlContent := strings.ReplaceAll(item.Description, "\n", "\n\n")

		if audioList[number].Number != number {
			panic("numers do not match")
		}

		episode := Episode{
			Number:      number,
			Title:       item.Title,
			Date:        t.Format("2006-01-02 15:04:05"),
			Guid:        item.Guid,
			Guest:       guest,
			Audio:       audioList[number].Name,
			Size:        audioList[number].Size,
			Description: strings.Split(htmlContent, "\n")[0],
			Duration:    item.Duration,
			Content:     htmlContent,
		}
		episodes = append(episodes, episode)
	}

	sort.Sort(Episodes(episodes))

	for i := 0; i < len(episodes); i++ {
		episode := episodes[i]

		files := []string{}
		err := filepath.Walk(episodeDir, func(path string, info os.FileInfo, err error) error {
			fullpath, err := filepath.Abs(path)
			files = append(files, fullpath)
			return nil
		})

		ut, err := template.New("episodes").Parse(layoutHeader)

		if err != nil {
			panic(err)
		}

		episodePath := filepath.Join(episodeDir, fmt.Sprintf("%v.md", episode.Number))

		os.Remove(episodePath)

		file, errCreate := os.Create(episodePath)
		if errCreate != nil {
			panic(errCreate)
		}

		fmt.Println("generate episode", i)

		err = ut.Execute(file, episode)

		if err != nil {
			panic(err)
		}
	}

	if isHugo {
		stdout, err := exec.Command("hugo").Output()
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
		fmt.Print(string(stdout))
		time.Sleep(1 * time.Second)
	}
}

func fetchSoundcloudRss(savePath string, url string) Rss {
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	os.Remove(savePath)

	saveFile, err := os.Create(savePath)

	if err != nil {
		panic(err)
	}
	defer saveFile.Close()

	rss := &Rss{}
	blob, err := io.ReadAll(resp.Body)

	_, err = saveFile.Write(blob)

	if err != nil {
		panic(err)
	}

	if err := xml.Unmarshal([]byte(blob), &rss); err != nil {
		panic(err)
	}
	return *rss
}

func fetchAudioList(audioS3Url string) []Audio {
	resp, err := http.Get(audioS3Url)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

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
		audio := Audio{
			Number: audioNumber,
			Name:   content.Key,
			Size:   content.Size,
		}
		audioList = append(audioList, audio)
	}

	sort.Sort(AudioList(audioList))

	return audioList
}

func getGuestName(audio string) (string, error) {
	parts := strings.Split(audio, ".")
	words := strings.Split(parts[0], "-")
	return fmt.Sprintf("%s-%s", words[len(words)-2], words[len(words)-1]), nil
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

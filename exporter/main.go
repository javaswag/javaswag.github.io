package main

import (
    "encoding/xml"
    "fmt"
    "log"
    "os"
    "io"
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
   XMLName  xml.Name `xml:"item"`
   Title       string `xml:"title"`
   Link        string `xml:"link"`
   Description string `xml:"description"`
   PubDate     string `xml:"pubDate"`
   Guid        string `xml:"guid"`
   Duration    string `xml:"duration"`
   Author      string `xml:"author"`
   Explicit    string `xml:"explicit"`
   Summary     string `xml:"summary"`
   Subtitle    string `xml:"subtitle"`
}

func main() {
    file, err := os.Open("./../layout/soundcloud_rss.xml")

    if err != nil {
         fmt.Println(err)
    }

    rss := &Rss{}
    blob, err := io.ReadAll(file)
    if err := xml.Unmarshal([]byte(blob), &rss); err != nil {
        log.Fatal(err)
    }

    for i := 0; i < len(rss.Channel.Items); i++ {
        item := rss.Channel.Items[i]
        fmt.Println(item.Title)
        fmt.Println(item.Duration)
    }
    


}

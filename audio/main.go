package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/go-mp3"
)

func main() {
	fmt.Println("Open: ", os.Args[1])
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	d, err := mp3.NewDecoder(f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("SampleRate %d\n", d.SampleRate())
	// 134,407,456
	fmt.Printf("Length %d bytes\n", d.Length())
	// fmt.Printf("Length %d Mb\n", )

}

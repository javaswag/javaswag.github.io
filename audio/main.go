package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/go-mp3"
)

const (
	gomp3NumChannels   = 2
	gomp3Precision     = 2
	gomp3BytesPerFrame = gomp3NumChannels * gomp3Precision
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

	fmt.Println(aMono)

	aData := aMono.RawData()
	aDataLen := len(aData)

	aDurationSecs := aDataLen / int(aMono.FrameRate())

	fmt.Println("aDurationSecs", aDurationSecs)

	aBandList := make([]int16, 0)

	BAND_STEP := int(math.Round(float64(aDataLen / bands)))

	fmt.Println("Analyzing audio..")

	for i := range bands {
		aBand := aData[i*BAND_STEP : (i+1)*BAND_STEP]
		// fmt.Println(aBand)
		aBandInfo := ABandInfo{}
		for index, sample := range aBand {
			absSample := int16(math.Abs(float64(sample)))
			aBandInfo.max = int16(math.Max(float64(absSample), float64(aBandInfo.max)))
			aBandInfo.avg = int16((absSample + aBandInfo.avg) / 2)
			if index < 5 {
				fmt.Printf("%d %v\n", index, aBandInfo)
			}
		}
		writeVal := aBandInfo.avg
		fmt.Println(i, writeVal)
		aBandList = append(aBandList, int16(writeVal))
	}
	// aBandList = append(aBandList, aDurationSecs)

	filename := fmt.Sprintf("{%s.avg%d.new.bin", f, bands)
	fmt.Println("Writing into: ", filename)

	fileOutput, err := os.Create(filename)
	if err != nil {
		fmt.Printf("%w", err)
	}

	defer fileOutput.Close()

	// d2 := []byte{115, 111, 109, 101, 10}

	// var i int16 = 41

	for i := range aBandList {
		arr := make([]byte, 0)
		binary.LittleEndian.PutUint16(arr, uint16(i))
		_, err := fileOutput.Write(arr)
		if err != nil {
			panic(err)
		}
	}
}

type ABandInfo struct {
	max int16
	avg int16
}

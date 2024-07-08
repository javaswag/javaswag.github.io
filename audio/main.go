package main

import (
	"fmt"
	"os"
	"path"
	"math"
	"github.com/hopesea/godub/v2"
)

const bands = 220

func main() {
	f := os.Args[1]
	fmt.Println("Open: ", f)
	filePath := path.Join(f)
	segment, err := godub.NewLoader().Load(filePath)
	if err != nil {
		fmt.Println("%w", err)
	}
	fmt.Println(segment)

	aMono, err := segment.ForkWithChannels(1)

	if err != nil {
		fmt.Println("%w", err)
	}

	fmt.Println(aMono)

	aData := aMono.RawData()
	aDataLen := len(aData)

	aDurationSecs := aDataLen / int(aMono.FrameRate())

	fmt.Println("aDurationSecs", aDurationSecs)

	aBandList := make([]int, 0)

	BAND_STEP := int(math.Round(float64(aDataLen / bands)))

	fmt.Println("Analyzing audio..")

	for i := range bands {
		aBand := aData[i * BAND_STEP:(i+1) * BAND_STEP]
		// fmt.Println(aBand)		aBandInfo := ABandInfo {}
		for sample := range aBand {
			absSample = int(math.Abs(sample))
			aBandInfo.max = int(math.Max(float64(absSample), float64(aBandInfo.max)))
			aBandInfo.avg = (absSample + aBandInfo.avg) / 2
		}

		writeVal := aBandInfo.avg
		fmt.Println(i, writeVal)
  		aBandList = append(aBandList, writeVal)
	}
	aBandList = append(aBandList, aDurationSecs)

	filename := fmt.Sprintf("{%s.avg%d.new.bin", f, bands)
	fmt.Println("Writing into: ", filename)

	fileOutput, err := os.Create(filename)
    if err != nil {
		fmt.Printf("%w", err)
	}

    defer fileOutput.Close()

    d2 := []byte{115, 111, 109, 101, 10}
    n2, err := fileOutput.Write(d2)
    if err != nil {
		fmt.Printf("%w", err)
	}
    fmt.Printf("wrote %d bytes\n", n2)
}

type ABandInfo struct {
	max int
	avg int
}

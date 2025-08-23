package internal

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hopesea/godub"
)

func Run(inputFile string, outputDir string) (string, error) {
	if !strings.HasSuffix(inputFile, ".mp3") {
		return "", fmt.Errorf("only mp3 files are supported")
	}

	// p := os.Args[1]
	// d := os.Args[2]
	fmt.Println("Open: ", inputFile)
	fmt.Println("Dir: ", outputDir)

	aMono, err := godub.NewLoader().Load(inputFile)

	if err != nil {
		log.Fatalf("%v", err)
	}

	aData := aMono.RawData()
	aDataLen := len(aData)

	aDurationSecs := aMono.Duration().Seconds()

	fmt.Printf("%s\n", aMono.String())

	aBandList := make([]float64, 0)

	bands := 220

	BAND_STEP := aDataLen / bands

	fmt.Printf("aDataLen %d\n", aDataLen)

	fmt.Printf("BAND_STEP %d\n", BAND_STEP)

	fmt.Println("Analyzing audio..")

	for i := range bands {
		aBand := aData[i*BAND_STEP : (i+1)*BAND_STEP]
		// fmt.Println(aBand)
		aBandInfo := ABandInfo{}
		for _, sample := range aBand {
			absSample := math.Abs(float64(sample))
			aBandInfo.max = math.Max(float64(absSample), float64(aBandInfo.max))
			aBandInfo.avg = (absSample + aBandInfo.avg) / 2
		}
		// writeVal := aBandInfo.avg
		writeVal := (aBandInfo.max - aBandInfo.avg) / 2
		fmt.Println(i, writeVal)
		aBandList = append(aBandList, writeVal)
	}
	aBandList = append(aBandList, aDurationSecs)

	filename := fmt.Sprintf("%s/%s.avg%d.bin", outputDir, filepath.Base(inputFile), bands)
	fmt.Println("Writing into: ", filename)

	err = os.Remove(filename)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	fileOutput, err := os.Create(filename)
	if err != nil {
		log.Fatalf("%w", err)
	}

	defer fileOutput.Close()

	var b = make([]byte, 2)
	for _, ii := range aBandList {
		binary.LittleEndian.PutUint16(b, uint16(ii))
		_, err := fileOutput.Write(b)
		if err != nil {
			fmt.Println("err!", err)
		}
		// fmt.Printf("%x ", b)
		b[0], b[1] = 0, 0
	}
	return filename, nil
}

type ABandInfo struct {
	max float64
	avg float64
}

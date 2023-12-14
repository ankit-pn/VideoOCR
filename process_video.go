package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"

	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
)

func imageToBytes(img image.Image) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func saveFrame(frame gocv.Mat, index int, videoname string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	client.SetLanguage("eng", "hin")

	img, err := frame.ToImage()
	if err != nil {
		return "", err
	}

	baseFolderPath := "content"
	subFolderName := videoname
	subFolderPath := filepath.Join(baseFolderPath, subFolderName)

	if err := os.MkdirAll(subFolderPath, os.ModePerm); err != nil {
		return "", err
	}

	fileName := "frame" + strconv.Itoa(index) + ".jpg"
	filePath := filepath.Join(subFolderPath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := jpeg.Encode(file, img, nil); err != nil {
		return "", err
	}

	imgBytes, err := imageToBytes(img)
	if err != nil {
		return "", err
	}

	if err := client.SetImageFromBytes(imgBytes); err != nil {
		return "", err
	}

	text, err := client.Text()
	if err != nil {
		return "", err
	}

	return text, nil
}

type File struct {
	FileID       string
	ParentFolder string
	FileData     []string
}

func processVideo(filename string) ([]string, error) {
	video, err := gocv.VideoCaptureFile(filename)
	if err != nil {
		return nil, err
	}
	defer video.Close()

	frame := gocv.NewMat()
	defer frame.Close()

	frameRate := int(video.Get(gocv.VideoCaptureFPS))
	frameCount := 0
	tenSecondsFrameInterval := frameRate * 10
	trunctedFilename := filepath.Base(filename)
	trunctedFilename = trunctedFilename[:len(trunctedFilename)-len(filepath.Ext(trunctedFilename))]
	var ocrArray []string

	for {
		if ok := video.Read(&frame); !ok {
			break
		}
		frameCount++

		if frameCount%tenSecondsFrameInterval == 0 {
			text, err := saveFrame(frame, frameCount/tenSecondsFrameInterval, trunctedFilename)
			if err != nil {
				fmt.Printf("Error processing frame %d of %s: %v\n", frameCount, filename, err)
				continue
			}
			ocrArray = append(ocrArray, text)
		}
	}

	return ocrArray, nil
}

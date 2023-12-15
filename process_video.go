package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
)

type File struct {
	FileID       string
	ParentFolder string
	FileData     []string
}

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

func processVideo(filePath string) ([]string, error) {
	log.Printf("Starting video processing: %s\n", filePath)
	var ocrValues []string

	video, err := gocv.VideoCaptureFile(filePath)
	if err != nil {
		return ocrValues, fmt.Errorf("error opening video file: %w", err)
	}
	defer video.Close()

	frameRate := video.Get(gocv.VideoCaptureFPS)
	if frameRate <= 0 {
		return ocrValues, fmt.Errorf("invalid frame rate for video: %s", filePath)
	}

	tenSecondInterval := int(frameRate) * 10
	if tenSecondInterval <= 0 {
		return ocrValues, fmt.Errorf("invalid interval calculated for video: %s", filePath)
	}

	frame := gocv.NewMat()
	defer frame.Close()

	frameCount := 0
	for {
		if ok := video.Read(&frame); !ok {
			log.Println("No more frames to read or error reading a frame")
			break
		}
		frameCount++

		if frameCount%tenSecondInterval == 0 {
			log.Printf("Processing frame %d", frameCount)
			text, err := saveFrame(frame, frameCount/tenSecondInterval, filepath.Base(filePath))
			if err != nil {
				log.Printf("Error processing frame %d: %v\n", frameCount, err)
				continue
			}
			ocrValues = append(ocrValues, text)
		}
	}

	return ocrValues, nil
}


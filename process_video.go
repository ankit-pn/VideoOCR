package main

import (
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"gocv.io/x/gocv"
    "github.com/otiai10/gosseract/v2"
    "bytes"
    "image"
)

func imageToBytes(img image.Image) []byte {
    buf := new(bytes.Buffer)
    err := jpeg.Encode(buf, img, nil)
    if err != nil {
        panic(err)
    }
    return buf.Bytes()
}

func saveFrame(frame gocv.Mat, index int,videoname string) (string,error) {
    client := gosseract.NewClient()
    defer client.Close()
    client.SetLanguage("eng", "hin")
    img, err := frame.ToImage()
    if err != nil {
        panic(err)
    }
    baseFolderPath := "content"  // Base folder path
    subFolderName := videoname
    subFolderPath := filepath.Join(baseFolderPath, subFolderName)

    err = os.MkdirAll(subFolderPath, os.ModePerm)
    if err != nil {
        panic(err)
    }
    fileName := "frame" + strconv.Itoa(index) + ".jpg"
    filePath := filepath.Join(subFolderPath, fileName) 
    fmt.Println(filePath)
    file, err := os.Create(filePath)
    if err != nil {
        panic(err)
    }
    defer file.Close()

    err = jpeg.Encode(file, img, nil)

      // Convert image.Image to []byte
    imgBytes := imageToBytes(img)

      // Set the image to the client from the []byte
    err = client.SetImageFromBytes(imgBytes)
	text, err := client.Text()
    if err != nil {
        panic(err)
    }
    fmt.Println(text)
    return text,err
}

type File struct {
	FileID string
	ParentFolder string
	FileData []string
}

func processVideo(filename string) ([]string,error) {
    video, err := gocv.VideoCaptureFile(filename)
    if err != nil {
        panic(err)
    }
    defer video.Close()

    frame := gocv.NewMat()
    defer frame.Close()

    frameRate := int(video.Get(gocv.VideoCaptureFPS))
    frameCount := 0
    tenSecondsFrameInterval := frameRate * 10 // 10 seconds interval
    filenameWithExtension := filepath.Base(filename)
    trunctedFilename := filenameWithExtension[:len(filenameWithExtension)-len(filepath.Ext(filenameWithExtension))]
    var ocrArray []string
    for {
        if ok := video.Read(&frame); !ok {
            break
        }
        frameCount++

        // Check if the current frame is at the 10 seconds interval
        if frameCount%tenSecondsFrameInterval == 0 {
            text,err1 := saveFrame(frame, frameCount/tenSecondsFrameInterval,trunctedFilename)
            if err1!=nil{
                fmt.Printf("Error while processing that frame %q : %v",filename,err)
            }
            ocrArray = append(ocrArray,text)
        }
    }
    return ocrArray,nil
}

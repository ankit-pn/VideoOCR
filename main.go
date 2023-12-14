package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"github.com/redis/go-redis/v9"
	"context"
	"encoding/json"
)


var ctx = context.Background()
var wg sync.WaitGroup


func GetKey(rdb *redis.Client, key string) (string, error) {
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key does not exist
	} else if err != nil {
		return "", err // Some other error
	}
	return val, nil
}

func SetKey(rdb *redis.Client, key string, value interface{}) error {
	// Convert your 'value' to a string that Redis can store
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	// fmt.Println(key)
	// Use the string(jsonData) as the value for the Redis set command
	return rdb.Set(ctx, key, string(jsonData), 0).Err()
}

func traverseDir(fileChan chan<- string, root_path string, rdb *redis.Client) {
	defer wg.Done()
	walkErr := filepath.Walk(root_path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error Accessing this path %q: %v\n", path, err)
			return err
		}
		fullFileName := info.Name()
		fileBaseName := strings.TrimSuffix(fullFileName, filepath.Ext(fullFileName))
		val, err := GetKey(rdb, fileBaseName)
		if err != nil {
			fmt.Printf("error checking key %q: %v\n", fileBaseName, err)
			return err // Handle the error as needed
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp4") && val == ""  {
            fileChan <- path
        }
        return nil
	})

	if walkErr != nil {
		fmt.Printf("Error Traversing the root Path")
	}
	

}


func ocrofimages(imageChan *string){
	// client := gosseract.NewClient()
	// defer client.Close()
	// client.SetLanguage("eng", "hin", "urd")
}



func workers(fileChan <-chan string,rdb *redis.Client,errChan chan<-error){
	defer wg.Done()
	for path := range fileChan{
		processVideo(path)
	}
}

func indexerEngine(folder_path string) {
	rdb:=redis.NewClient(&redis.Options{
		Addr : "localhost:6379",
		Password: "",
		DB: 1,
	})
	root_path := "/home/kg766/mnt/kg766/WhatsappMonitorData/downloaded-media"
	buffSize := 500
	fileChan := make(chan string, buffSize)
	
	wg.Add(1)
	go traverseDir(fileChan,root_path,rdb)
	// go processVideo(fileChan)

}

func main() {

	x:=processVideo("")
	fmt.Println(x)
	// fmt.Println(folder_path)

}

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
	"github.com/joho/godotenv"
	"log"
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
	defer close(fileChan)
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






func worker(fileChan <-chan string,rdb *redis.Client,ctx context.Context,errChan chan<-error){
	defer wg.Done()
	for path := range fileChan{
		ocrValue,err := processVideo(path)
		if err != nil {
			errChan <- fmt.Errorf("error extracting text from %q: %w", path, err)
			continue
		}
		parentFolder := filepath.Dir(path)
		filenameWithExtension := filepath.Base(path)
    	trunctedFilename := filenameWithExtension[:len(filenameWithExtension)-len(filepath.Ext(filenameWithExtension))]
		file := File{
			FileID: trunctedFilename,
			ParentFolder: parentFolder,
			FileData: ocrValue,
		}
		err = SetKey(rdb, trunctedFilename, file)
		if err != nil {
			fmt.Println("Failed to set value in Redis:", err)
			return
		}
	}
}

func indexerEngine(root_path string) {
	rdb:=redis.NewClient(&redis.Options{
		Addr : "localhost:6379",
		Password: "",
		DB: 1,
	})
	

	buffSize := 25
	fileChan := make(chan string, buffSize)
	errChan := make(chan error, buffSize)
	for i := 0; i < 2; i++ {
        wg.Add(1)
        go worker(fileChan, rdb, ctx,errChan)
    }


	wg.Add(1)
	go traverseDir(fileChan,root_path,rdb)
	
	wg.Wait()
	
	close(errChan) 
    for err := range errChan {
		fmt.Println("Error from worker:", err)
	}

	// go processVideo(fileChan)

}


func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
}

func main() {
	rootPath := os.Getenv("ROOT_PATH")
	if rootPath == "" {
		log.Fatal("Root Path Not Set")
	}
	indexerEngine(rootPath)
}

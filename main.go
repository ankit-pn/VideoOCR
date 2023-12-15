package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var wg sync.WaitGroup



func GetKey(rdb *redis.Client, key string) (string, error) {
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		log.Printf("Key does not exist: %s\n", key)
		return "", nil // Key does not exist
	} else if err != nil {
		log.Printf("Error getting key %s: %v\n", key, err)
		return "", err // Some other error
	}
	return val, nil
}

func SetKey(rdb *redis.Client, key string, value interface{}) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		log.Printf("Error marshalling value for key %s: %v\n", key, err)
		return err
	}
	err = rdb.Set(ctx, key, string(jsonData), 0).Err()
	if err != nil {
		log.Printf("Failed to set value in Redis for key %s: %v\n", key, err)
	}
	return err
}

func worker(fileChan <-chan string, rdb *redis.Client, ctx context.Context, errChan chan<- error) {
	defer wg.Done()
	for path := range fileChan {
		log.Printf("Processing file: %s\n", path)
		ocrValue, err := processVideo(path) // Ensure processVideo is defined and works correctly
		if err != nil {
			log.Printf("Error extracting text from %q: %v\n", path, err)
			errChan <- fmt.Errorf("error extracting text from %q: %w", path, err)
			continue
		}
		parentFolder := filepath.Dir(path)
		filenameWithExtension := filepath.Base(path)
		truncatedFilename := filenameWithExtension[:len(filenameWithExtension)-len(filepath.Ext(filenameWithExtension))]
		file := File{
			FileID:       truncatedFilename,
			ParentFolder: parentFolder,
			FileData:     ocrValue,
		}
		err = SetKey(rdb, truncatedFilename, file)
		if err != nil {
			log.Printf("Failed to set value in Redis for file %s: %v\n", truncatedFilename, err)
			return
		}
	}
}

func indexerEngine(root_path string) {
	log.Println("Initializing Redis client...")
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	buffSize := 12
	fileChan := make(chan string, buffSize)
	errChan := make(chan error, buffSize)
	for i := 0; i < buffSize; i++ {
		wg.Add(1)
		go worker(fileChan, rdb, ctx, errChan)
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	walkErr := filepath.Walk(root_path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp4") {
			fileBaseName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			val, err := GetKey(rdb, fileBaseName)
			if err != nil {
				log.Printf("Error checking key %q: %v\n", fileBaseName, err)
				return err // Handle the error as needed
			}
			if val == "" {
				fileChan <- path
			}
		}
		return nil
	})

	if walkErr != nil {
		log.Printf("Error traversing the root path: %v\n", walkErr)
	}
	close(fileChan)

	for err := range errChan {
		log.Printf("Error from worker: %v\n", err)
	}
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
}

func main() {
	log.Println("Starting main function...")
	rootPath := os.Getenv("ROOT_PATH")
	if rootPath == "" {
		log.Fatal("Root Path Not Set")
	}
	log.Printf("Root path set to: %s\n", rootPath)
	indexerEngine(rootPath)
}

// processVideo function needs to be defined

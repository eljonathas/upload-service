package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/joho/godotenv"
)

type DeployRequest struct {
	RepoUrl string `json:"repo_url"`
}

func randomId(length int) string {
	const charset string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func getFiles(path string) ([]string, error) {
	var files []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func uploadToS3(path string) {
	s3BucketName := os.Getenv("S3_BUCKET")
	s3KeyName := os.Getenv("S3_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		log.Fatal(err)
	}

	client := s3.NewFromConfig(cfg)

	files, err := getFiles(path)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}

		_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: &s3BucketName,
			Key:    &s3KeyName,
			Body:   f,
		})

		if err != nil {
			log.Fatal(err)
		}
	}

}

func deploy(c *gin.Context) {
	var deployRequest DeployRequest
	deployId := randomId(6)

	outputDir := "./output/" + deployId

	if err := c.BindJSON(&deployRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := git.PlainClone(outputDir, false, &git.CloneOptions{
		URL:      deployRequest.RepoUrl,
		Progress: os.Stdout,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Error cloning repository",
			"error":   err.Error(),
		})
		return
	}

	uploadToS3(outputDir)

	log.Println(deployRequest.RepoUrl)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Error cloning repository",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deployId": deployId,
	})
}

func main() {
	godotenv.Load(".env")
	router := gin.Default()
	router.POST("/deploy", deploy)
	router.Run("localhost:8080")
}

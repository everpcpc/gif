package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	filePath = "tmp"
	filter   = "fps=15"
)

func main() {
	router := gin.New()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB (default is 32 MiB)
	router.Static("/css", "./css")
	router.Static("/js", "./js")
	router.LoadHTMLGlob("templates/*")
	router.GET("/upload", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", gin.H{
			"title": "File upload",
		})
	})
	router.POST("/upload", upload)
	router.Run(":8080")
}

func upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("input err: %s", err.Error()))
		return
	}

	tmpPath := path.Join(filePath, strconv.FormatInt(time.Now().UnixNano(), 10))
	if err := os.MkdirAll(tmpPath, os.ModePerm); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("create dir err: %s", err.Error()))
		return
	}
	defer os.RemoveAll(tmpPath)

	fullFileName := path.Join(tmpPath, file.Filename)
	tmpFileName := path.Join(tmpPath, "pattern.png")
	token := strings.Split(file.Filename, ".")
	token[len(token)-1] = "gif"
	targetFileName := strings.Join(token, ".")
	targetFile := path.Join(tmpPath, targetFileName)

	if err := c.SaveUploadedFile(file, fullFileName); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return
	}

	patternArgs := []string{
		"-v", "warning",
		"-i", fullFileName,
		"-vf", filter + ",palettegen",
		"-y", tmpFileName,
	}
	patternCmd := exec.Command("ffmpeg", patternArgs...)
	if err := patternCmd.Run(); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("pattern err: %s", err.Error()))
		return
	}

	convertArgs := []string{
		"-v", "warning",
		"-i", fullFileName,
		"-i", tmpFileName,
		"-lavfi", filter + " [x]; [x][1:v] paletteuse",
		"-y", targetFile,
	}
	convertCmd := exec.Command("ffmpeg", convertArgs...)
	if err := convertCmd.Run(); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("convert err: %s", err.Error()))
		return
	}

	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+targetFileName)
	c.Header("Content-Type", "application/octet-stream")
	c.File(targetFile)
}

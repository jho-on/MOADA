package main

import (
	"fmt"

	"strings"

	"net/http"

	"github.com/gin-gonic/gin"

	"os"

	"github.com/joho/godotenv"

	
	"backend/utils"
	"backend/db"

	"path/filepath"
)



var allowedTypes = map[string]bool{
    "image/jpeg":                true,
    "image/jpg":                 true,
    "image/png":                 true,
    "image/gif":                 true,

    "application/pdf":           true,
    "application/json":          true,

    "application/zip":           true,
    "application/x-tar":         true,
    "application/x-rar-compressed": true,

    "text/plain":               true,

    "audio/mpeg":                true,
    "audio/x-wav":               true,
    "audio/x-flac":              true,
}


func saveFile(c *gin.Context) {
	receivedFile, err := c.FormFile("file")
	email := c.DefaultPostForm("email", "")
	fmt.Printf("%v\n", receivedFile)
	typeFile := receivedFile.Header.Get("Content-Type")

	allowed := allowedTypes[typeFile]

	

	ip := c.Request.Header.Get("CF-Connecting-IP")
    if ip == "" {
        ip = c.ClientIP()
    }

	// File Validation
	if err != nil || receivedFile == nil{
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Error receiving file.",
		})
		return
	}
	
	// Extension validation
	if !allowed {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The uploaded file is not allowed. You can try compressing it in .rar, .zip, or .tar format, for example.",
		})
		return
	}
	
	
	// Testing Virus
	path_ := filepath.Join(os.TempDir(), strings.Split(receivedFile.Filename, ".")[0])
    if err = c.SaveUploadedFile(receivedFile, path_); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "erro": "Error saving the file temporarily",
        })
        return
    }
    defer os.Remove(path_) 
	
	
	hasVirus, err := utils.CheckVirus(path_)
	if err != nil && !hasVirus {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error checking for viruses in the file.",
		})
		return
	}
	
	if hasVirus {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The system detected the file as infected with a virus.",
		})
		return
	}


	content, err := os.ReadFile(path_)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error accessing file contents.",
		})
		return
	}
	
	// Validating Email
	if !utils.ValidateEmail(email) && email != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The uploaded file is not allowed. You can try compressing it in .rar, .zip, or .tar format, for example.",
		})
		return
	}
	
	// Saving File
	var newFile db.File
	idDownload := (utils.EncryptString(string(content) + receivedFile.Filename) + string(os.Getenv("ENCRYPTION_KEY")))
	idDelete := (utils.EncryptString(string(content) + receivedFile.Filename) + string(os.Getenv("ENCRYPTION_KEY")) + string(os.Getenv("EXCLUSION_KEY")))
	filePath := filepath.Join(os.Getenv("SAVE_PATH") + string(utils.EncryptString(string(ip))), utils.EncryptString(string(idDownload)) + "." + strings.Split(receivedFile.Filename, ".")[1])

	if utils.FileExists(filePath) {
		existingFile, srcErr := db.GetFileFromID(utils.EncryptString(string(idDownload)))
	
		if srcErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "The file is already on the server, but it could not be found. " + srcErr.Error(),
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "The file is already on the server.",
				"data":  existingFile,
			})
		}
	
		return
	}
	
	
	newFile, err = db.SaveMetadata(idDownload, idDelete, receivedFile.Filename, email, float64(receivedFile.Size))
	
	if err != nil {
		fmt.Printf("%v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": err,
		})
		return
	}

	dirPath := filepath.Dir(filePath)
	err = os.MkdirAll(dirPath, os.ModePerm)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error creating directory.",
		})
		return
	}
	fmt.Printf("%v\n", filePath)
    if err := c.SaveUploadedFile(receivedFile, filePath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"erro": "Error while saving file"})
        return
    }

	
	if saveUser(ip, c) {
		c.JSON(http.StatusOK, gin.H{
			"message": "File saved successfully",
			"data": newFile,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create new user",
		})
	}	
}

func deleteFile(c *gin.Context){
	idDelete := c.DefaultQuery("idDelete", "0")
	ip := c.Request.Header.Get("CF-Connecting-IP")
    if ip == "" {
        ip = c.ClientIP()
    }

	if idDelete == "0" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The provided id is not valid",
		})
		return
	}
	
	file, err := db.DeleteFile(idDelete)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	
	path_ := filepath.Join(os.Getenv("SAVE_PATH") + string(utils.EncryptString(string(ip))), string(file.IdDownload) + "." + strings.Split(file.Name, ".")[1])
	
	err = os.Remove(path_)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error deleting file",
		})
		return
	}
	
	

	if saveUser(ip, c) {
		c.JSON(http.StatusOK, gin.H{
			"message": "File deleted successfully",
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user data",
		})
	}	
	
}

func downloadFile(c *gin.Context){
	idDownload := c.DefaultQuery("idDownload", "0")
	ip := c.Request.Header.Get("CF-Connecting-IP")
    if ip == "" {
        ip = c.ClientIP()
    }

	if idDownload == "0" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The provided id is not valid",
		})
		return
	}
	
	file, err := db.GetFileFromID(idDownload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error retrieving file from the server",
		})
		return
	}
	
	path_ := filepath.Join(os.Getenv("SAVE_PATH") + string(utils.EncryptString(string(ip))), string(file.IdDownload) + "." + strings.Split(file.Name, ".")[1])
	
	if !utils.FileExists(path_) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "File not found",
		})
		return
	}
	
	c.File(path_)
}

func saveUser(ip string, c *gin.Context) (bool){

	if !db.UserExists(utils.EncryptString(ip)) {
		
		newUser, err := db.CreateUser(utils.EncryptString(ip), filepath.Join(os.Getenv("SAVE_PATH") + string(utils.EncryptString(ip))))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"erro": err,
			})
			return false
		}
		
		c.JSON(http.StatusOK, gin.H{
			"message": "User created successfully",
			"user":    newUser,
		})
		
	
	}else{
		err := db.UpdateUser(utils.EncryptString(ip), filepath.Join(os.Getenv("SAVE_PATH") + string(utils.EncryptString(ip))))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"erro": err,
			})
			return false
		}
	}
	return true
}



func main(){
	
	if err := godotenv.Load(); err != nil {
		fmt.Print("Error loading .env" + err.Error())
		return
	}

	db.Connect(os.Getenv("DB_URI"))

	router := gin.Default()
	router.POST("/sendFile", saveFile)
	router.GET("/deleteFile", deleteFile)
	router.GET("/downloadFile", downloadFile)

	router.Run(":" + os.Getenv("PORT"))
}
package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"os"

	"github.com/joho/godotenv"

	"github.com/gin-contrib/cors"

	"backend/db"
	"backend/utils"

	"path/filepath"
	"time"

	"log"
)

const (
	requestLimit      = 5
	timeLimit         = 1 * time.Minute
	userMaxSpace      = float64(75 * (1024 * 1024)) // 75MB per user
	maxHostSpaceUsage = 68 * userMaxSpace           // around 5GB, 68 users
)

var logFile *os.File
var allowedTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,

	"application/pdf":  true,
	"application/json": true,

	"application/zip":              true,
	"application/x-tar":            true,
	"application/x-rar-compressed": true,

	"text/plain": true,

	"audio/mpeg":   true,
	"audio/x-wav":  true,
	"audio/x-flac": true,
}

func init() {
	var err error
	logFile, err = os.OpenFile("requisitions.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
}

func saveFile(c *gin.Context) {
	receivedFile, err := c.FormFile("file")
	email := c.DefaultPostForm("email", "")
	typeFile := receivedFile.Header.Get("Content-Type")

	allowed := allowedTypes[typeFile]

	if utils.GetFolderSize(os.Getenv("SAVE_PATH")) >= maxHostSpaceUsage {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "The host server storage capacity is full.",
		})
		return
	}

	ip := c.Request.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = c.ClientIP()
	}

	// File Validation
	if err != nil || receivedFile == nil {
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

	if _, err := db.GetUser(string(utils.EncryptString(ip))); err == nil {
		err = rateLimit(string(utils.EncryptString(ip)))
		fmt.Printf("%v\n", err)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"erro": "error related to ratelimit",
			})
			return
		}
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

	// Check available space
	user, err := db.GetUser(utils.EncryptString(ip))
	if err == nil {
		if float64(user.UsedSpace)+float64(receivedFile.Size) > userMaxSpace {
			remainingSpace := float64((userMaxSpace - float64(user.UsedSpace)) / (1024 * 1024))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("The file size exceeds your available storage capacity. You have %.2f MB left.", remainingSpace),
			})
			return
		}
	}

	// File content reading (to encrypt for ID generation)
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

	// Generate ID for the file
	var newFile db.File
	idPublic := (utils.EncryptString(string(content)+receivedFile.Filename) + string(os.Getenv("ENCRYPTION_KEY")))
	idPrivate := (utils.EncryptString(string(content)+receivedFile.Filename) + string(os.Getenv("ENCRYPTION_KEY")) + string(os.Getenv("EXCLUSION_KEY")))
	filePath := filepath.Join(os.Getenv("SAVE_PATH")+string(utils.EncryptString(string(ip))), utils.EncryptString(string(idPublic))+"."+strings.Split(receivedFile.Filename, ".")[1])

	// Check if the file already exists
	if utils.FileExists(filePath) {
		existingFile, srcErr := db.GetFileFromID(utils.EncryptString(string(idPublic)), "public")

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

	// Save metadata to the DB
	newFile, err = db.SaveMetadata(idPublic, idPrivate, receivedFile.Filename, email, float64(receivedFile.Size))
	if err != nil {
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

	tempFilePath := filepath.Join(os.TempDir(), "temp_"+receivedFile.Filename)
	if err := c.SaveUploadedFile(receivedFile, tempFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "Error while saving temporary file"})
		return
	}
	defer os.Remove(tempFilePath)

	if err := utils.CopyFileWithoutMetadata(tempFilePath, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "Error processing the file"})
		return
	}

	if saveUser(ip, c) {
		c.JSON(http.StatusOK, gin.H{
			"message": "File saved successfully",
			"data":    newFile,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create new user",
		})
	}
}

func deleteFile(c *gin.Context) {
	idPrivate := c.DefaultPostForm("idPrivate", "0")
	ip := c.Request.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = c.ClientIP()
	}

	if idPrivate == "0" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The provided id is not valid",
		})
		return
	}

	file, err := db.DeleteFile(idPrivate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}

	path_ := filepath.Join(os.Getenv("SAVE_PATH")+string(utils.EncryptString(string(ip))), string(file.IdPublic)+"."+strings.Split(file.Name, ".")[1])

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

func downloadFile(c *gin.Context) {
	idPublic := c.DefaultQuery("idPublic", "0")
	ip := c.Request.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = c.ClientIP()
	}

	if idPublic == "0" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The provided id is not valid",
		})
		return
	}

	file, err := db.GetFileFromID(idPublic, "public")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error retrieving file from the server",
		})
		return
	}
	fmt.Printf("%v", file.Name)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Name))

	path_ := filepath.Join(os.Getenv("SAVE_PATH")+string(utils.EncryptString(string(ip))), string(file.IdPublic)+"."+strings.Split(file.Name, ".")[1])

	if !utils.FileExists(path_) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "File not found",
		})
		return
	}

	c.File(path_)
}

func saveUser(ip string, c *gin.Context) bool {

	if !db.UserExists(utils.EncryptString(ip)) {

		_, err := db.CreateUser(utils.EncryptString(ip), filepath.Join(os.Getenv("SAVE_PATH")+string(utils.EncryptString(ip))))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"erro": err,
			})
			return false
		}

		return true
		// c.JSON(http.StatusOK, gin.H{
		// 	"message": "User created successfully",
		// 	"user":    newUser,
		// })

	} else {
		err := db.UpdateUser(utils.EncryptString(ip), filepath.Join(os.Getenv("SAVE_PATH")+string(utils.EncryptString(ip))))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"erro": err,
			})
			return false
		}
	}
	return true
}

func userInfo(c *gin.Context) {
	ip := c.Request.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = c.ClientIP()
	}

	user, err := db.GetUser(string(utils.EncryptString(ip)))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": err,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

func fileInfo(c *gin.Context) {
	idPrivate := c.DefaultQuery("idPrivate", "0")

	if idPrivate == "0" {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "The 'idPrivate' parameter was not provided or is invalid.",
		})
		return
	}

	file, err := db.GetFileFromID(idPrivate, "private")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": err,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": file,
	})
}

func deleteUser(c *gin.Context) {
	ip := c.Request.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = c.ClientIP()
	}

	err := db.DeleteUser(string(utils.EncryptString(ip)))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": err,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All of your data has been erased",
	})
}

func rateLimit(ip string) error {
	user, err := db.GetUser(ip)

	if err != nil {
		return err
	}

	if time.Since(user.APILastCallDate) > timeLimit {
		err = db.ResetRateLimit(ip)
		if err != nil {
			return err
		}
	}

	if user.APICalls >= requestLimit {
		return fmt.Errorf("the number of API calls has been exceeded")
	}

	err = db.UpdateAPIRelatedData(ip)

	if err != nil {
		return err
	}

	return nil
}

func logUnauthorizedRequests() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != os.Getenv("ALLOWED_ORIGIN") {
			logMessage := fmt.Sprintf("[%s] Unauthorized request from: %s - Method: %s - Path: %s\n",
				time.Now().Format(time.RFC3339),
				origin,
				c.Request.Method,
				c.Request.URL.Path,
			)

			if _, err := logFile.WriteString(logMessage); err != nil {
				log.Printf("Error writing to the log file: %v", err)
			}
		}
		c.Next()
	}
}

func main() {

	if err := godotenv.Load(); err != nil {
		fmt.Print("Error loading .env" + err.Error())
		return
	}

	db.Connect(os.Getenv("DB_URI"))

	router := gin.Default()

	router.Use(logUnauthorizedRequests())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("ALLOWED_ORIGIN")},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
	}))

	router.POST("/sendFile", saveFile) //
	router.POST("/deleteFile", deleteFile)
	router.POST("/deleteUser", deleteUser)

	router.GET("/downloadFile", downloadFile) //
	router.GET("/myInfo", userInfo)           //
	router.GET("/fileInfo", fileInfo)         //

	err := router.Run(":" + os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("Fail in server init: %v", err)
	}
}

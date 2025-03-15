package db

import (
	"context"
	"strings"

	"fmt"

	"time"

	"backend/utils"
	"path/filepath"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/joho/godotenv"
	"os"
)

type File struct {
	IdDownload string `json:"idDownload" bson:"idDownload"`
	IdDelete string `json:"idDelete" bson:"idDelete"`
	Name string `json:"name" bson:"name"`
	Size float64 `json:"size" bson:"size"`
	SavedDate time.Time `json:"savedDate" bson:"savedDate"`
	ExpireDate time.Time `json:"expireDate" bson:"expireDate"`
	Email string `json:"email" bson:"email"`
}

type User struct {
	Ip string `json:"ip" bson:"ip"`
	Files []File `json:"files" bson:"files"`
	FilesNumber int `json:"filesNumber" bson:"filesNumber"`
	UsedSpace float64 `json:"usedSpace" bson:"usedSpace"`
	IpSavedDate time.Time `json:"ipSavedDate" bson:"ipSavedDate"`
	IpExpireDate time.Time `json:"ipExpireDate" bson:"ipExpireDate"`
	APICalls int `json:"APICalls" bson:"APICalls"`
}

var client *mongo.Client
var collection *mongo.Collection

func Connect(uri string) {
	if client != nil {
		return 
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Printf("Error connecting to MongoDB: %v", err)
		return
	}


	err = client.Ping(ctx, nil)
	if err != nil {
		fmt.Printf("Error pinging MongoDB: %v\n", err)
		return
	}

	fmt.Printf("Successfully connected to MongoDB\n")
}


func ChangeCollection(dbName, collectionName string) {
	collection = client.Database(dbName).Collection(collectionName)
}

func SaveMetadata(idDownload, idDelete, name, email string, size float64) (File, error) {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("FILES_COLLECTION"))
	newFile := File{
		IdDownload: utils.EncryptString(idDownload),
		IdDelete: utils.EncryptString(idDelete),
		Name: name,
		Size: size,
		SavedDate:  time.Now(),
		ExpireDate: time.Now().AddDate(0, 0, 1),
		Email: email,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, newFile)



	if err != nil {
		return File{}, fmt.Errorf("error while saving the metadata")
	}
	
	return newFile, nil
}

func GetFileFromID(idDownload string) (File, error){
 	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("FILES_COLLECTION"))
	var file File

	filter := bson.M{"idDownload": idDownload}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err := collection.FindOne(ctx, filter).Decode(&file)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return File{}, fmt.Errorf("no document found with the specified idDownload")
		} else {
			return File{}, fmt.Errorf("error retrieving the file: %v", err)
		}
	}

	return file, nil
}

func DeleteFile(idDelete string) (File, error){
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("FILES_COLLECTION"))
	filter := bson.D{{Key: "idDelete", Value: idDelete}}
	var res bson.M

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, filter).Decode(&res)
	if err != nil {
		return File{}, fmt.Errorf("error retrieving file from the database")
	}

	idDownload, ok := res["idDownload"].(string)
	if !ok {
		return File{}, fmt.Errorf("field 'idDownload' not found in the database")
	}

	file, err := GetFileFromID(idDownload)
	if err != nil {
		return File{}, fmt.Errorf("error retrieving file metadata")
	}

	result, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return File{}, fmt.Errorf("error attempting to delete the file from the database")
	}

	if result.DeletedCount < 1 {
		return File{}, fmt.Errorf("the file was not deleted from the database")
	}


	return file, nil
}

func UserExists(ip string) bool {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	filter := bson.D{{Key: "ip", Value: ip}}


	var result bson.M
	err := collection.FindOne(context.Background(), filter).Decode(&result)


	if err == mongo.ErrNoDocuments {
		return false 
	}

	if err != nil {
		return false
	}

	return true
}

func CreateUser(ip string, DirPath string) (User, error){
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	filesNumber, usedSpace, files, err := getFilesFromDir(DirPath)
	
	if err != nil {
		return User{}, err
	}
	fmt.Printf("%v\n" , files)


	newUser := User {
		Ip: ip,
		Files: files,
		FilesNumber: filesNumber,
		UsedSpace: usedSpace,
		IpSavedDate: time.Now(),
		IpExpireDate: time.Now().AddDate(0, 0, 1),
		APICalls: 1,
	}


	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a, err := collection.InsertOne(ctx, newUser)
	fmt.Printf("%v\n%v\n", err, a)
	if err != nil {
		
		return User{}, fmt.Errorf("error while creating user")
	}
	
	

	return newUser, nil
}

func UpdateUser(ip string, DirPath string) (error){
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	filesNumber, usedSpace, files, err := getFilesFromDir(DirPath)
	
	if err != nil {
		return err
	}

	filter := bson.D{{Key: "ip", Value: ip}}


	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "files", Value: files},
			{Key: "filesNumber", Value: filesNumber},
			{Key: "usedSpace", Value: usedSpace},
			{Key: "ipExpireDate", Value: time.Now().AddDate(0, 0, 1)},
		}},
	}

	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return fmt.Errorf("error while updating user data")
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user was not updated")
	}

	update = bson.D{
		{Key: "$inc", Value: bson.D{
			{Key: "APICalls", Value: 1}, 
		}},
	}

	result, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return fmt.Errorf("error while updating user data")
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user was not updated")
	}


	return nil
}

func getFilesFromDir(DirPath string) (int, float64, []File, error){
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("FILES_COLLECTION"))
	godotenv.Load()

	filesArray, err := utils.GetStoredFiles(DirPath)
	var filesNumber int
	var usedSpace float64
	var files []File

	if err != nil {
		return 0, 0.0, []File{}, fmt.Errorf("error accessing user's directory")
	}

	
	for _, arq := range filesArray {
		idDownload := strings.Split((filepath.Base(arq)), ".")[0]
		fileNow, err := GetFileFromID(idDownload)
		if err != nil {
			return 0, 0.0, []File{}, fmt.Errorf("error retrieving file from ID %s: %v", idDownload, err)
		}

		files = append(files, fileNow)
		filesNumber += 1
		usedSpace += fileNow.Size
	}
	
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	return filesNumber, usedSpace, files, nil
}

// Package db handles database operations for managing users and their associated files, using MongoDB.
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

	"os"

	"github.com/joho/godotenv"
)

// File represents a file uploaded by a user with metadata such as identifiers, name, size, and associated email.
type File struct {
	IdPublic string `json:"idPublic" bson:"idPublic"` // Public identifier of the file
	IdPrivate string `json:"idPrivate" bson:"idPrivate"` // Private identifier of the file
	Name string `json:"name" bson:"name"` // Name of the file
	Size float64 `json:"size" bson:"size"` // Size of the file in bytes
	SavedDate  time.Time `json:"savedDate" bson:"savedDate"` // Date when the file was saved
	ExpireDate time.Time `json:"expireDate" bson:"expireDate"` // Expiration date of the file
	Email string `json:"email" bson:"email"` // Email of the user who uploaded the file
}


// User represents a user in the system.
// It contains information about the users anonymized (hashed) IP address, file data, and metadata for usage tracking.
type User struct {
    Ip string `bson:"ip"`  // anonymized (hashed) IP address of the user
    Files []string `bson:"files"`  // List of public ids for files associated with the user
    FilesNumber int `bson:"filesNumber"`  // Number of files the user has uploaded
    UsedSpace float64 `bson:"usedSpace"`  // Total space consumed by the user
    IpSavedDate time.Time `bson:"ipSavedDate"`  // Date when the users IP was saved
    IpExpireDate time.Time `bson:"ipExpireDate"` // Expiration date for the users data
    APICalls int `bson:"APICalls"` // Number of API calls made by the user
    APILastCallDate time.Time `bson:"APILastCallDate"`  // Date of the last API call
}


var client *mongo.Client
var collection *mongo.Collection


// Connect establishes a connection to a MongoDB database.
// Parameters:
//   uri (string): The URI connection string for the MongoDB database.
// Returns:
//   None: This function does not return any values. It prints messages to indicate the connection status.
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

// ChangeCollection switches the current MongoDB collection to the one specified by the provided database and collection names.
// Parameters:
//   dbName (string): The name of the database to connect to.
//   collectionName (string): The name of the collection within the database.
// Returns:
//   None: This function does not return any values. It changes the current collection.
func ChangeCollection(dbName, collectionName string) {
	collection = client.Database(dbName).Collection(collectionName)
}

// SaveMetadata saves file metadata to the MongoDB collection.
// Parameters:
//   idPublic (string): The public ID of the file.
//   idPrivate (string): The private ID of the file.
//   name (string): The name, with extension, of the file.
//   email (string): The email associated with the file.
//   size (float64): The size of the file in bytes.
// Returns:
//   File: The saved File object.
//   error: An error if there was an issue saving the metadata.
func SaveMetadata(idPublic, idPrivate, name, email string, size float64) (File, error) {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("FILES_COLLECTION"))
	newFile := File{
		IdPublic: utils.EncryptString(idPublic),
		IdPrivate:   utils.EncryptString(idPrivate),
		Name:       name,
		Size:       size,
		SavedDate:  time.Now(),
		ExpireDate: time.Now().AddDate(0, 0, 1),
		Email:      email,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, newFile)

	if err != nil {
		return File{}, fmt.Errorf("error while saving the metadata")
	}

	return newFile, nil
}

// GetFileFromID retrieves a file from the MongoDB collection based on the provided ID and ID type.
// Parameters:
//   id (string): The ID of the file to retrieve (either public or private).
//   idType (string): The type of ID provided. It can either be "public" or "private".
// Returns:
//   File: The file object retrieved from the database.
//   error: An error if there was an issue retrieving the file or if no document is found.
func GetFileFromID(id, idType string) (File, error) {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("FILES_COLLECTION"))
	var file File
	var filter bson.M

	if idType == "private"{
		filter = bson.M{"idPrivate": id}
	}else if idType == "public"{
		filter = bson.M{"idPublic": id}
	}else{
		return File{}, fmt.Errorf("idType provided not valid")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, filter).Decode(&file)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return File{}, fmt.Errorf("no document found with the specified id")
		} else {
			return File{}, fmt.Errorf("error retrieving the file: %v", err)
		}
	}

	return file, nil
}

// DeleteFile deletes a file from the MongoDB collection based on its private ID.
// Parameters:
//   idPrivate (string): The private ID of the file to delete.
// Returns:
//   File: The file object that was deleted.
//   error: An error if there was an issue.
func DeleteFile(idPrivate string) (File, error) {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("FILES_COLLECTION"))
	filter := bson.D{{Key: "idPrivate", Value: idPrivate}}
	var res bson.M

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, filter).Decode(&res)
	if err != nil {
		return File{}, fmt.Errorf("error retrieving file from the database")
	}

	IdPublic, ok := res["idPublic"].(string)
	if !ok {
		return File{}, fmt.Errorf("field 'IdPublic' not found in the database")
	}

	file, err := GetFileFromID(IdPublic, "public")
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

// UserExists checks if a user with a specific anonymized (hashed) IP address exists in the MongoDB collection.
// Parameters:
//   ip (string): The IP address anonymized (hashed) to search for in the users collection.
// Returns:
//   bool: Returns true if the user exists, false otherwise.
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

// CreateUser creates a new user in the MongoDB collection and returns the created user or an error.
// Parameters:
//   ip (string): The IP address of the user to create.
//   DirPath (string): The path to the directory containing the users files.
// Returns:
//   User: The created user object if successful.
//   error: An error if the user creation fails.
func CreateUser(ip string, DirPath string) (User, error) {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	
	filesNumber, usedSpace, ids, err := getFilesFromDir(DirPath)
	
	if err != nil {
		return User{}, err
	}
	
	newUser := User{
		Ip: ip,
		Files: ids,
		FilesNumber: filesNumber,
		UsedSpace: usedSpace,
		IpSavedDate: time.Now(),
		IpExpireDate: time.Now().AddDate(0, 0, 1),
		APICalls: 1,
		APILastCallDate: time.Now(),
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

// UpdateUser updates a users data based on their anonymized (hashed) IP address and the directory path with new file information.
// Parameters:
//   ip (string): The anonymized (hashed) IP address of the user whose data is being updated.
//   DirPath (string): The path to the directory containing the users files.
// Returns:
//   error: Returns nil if the update is successful, or an error message if something goes wrong.
func UpdateUser(ip string, DirPath string) error {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	filesNumber, usedSpace, ids, err := getFilesFromDir(DirPath)
	
	if err != nil {
		return err
	}

	filter := bson.D{{Key: "ip", Value: ip}}

	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "files", Value: ids},
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

	return nil
}

// UpdateAPIRelatedData increments the anonymized (hashed) API call count and updates the last API call timestamp for a user.
// Parameters:
//   ip (string): The anonymized (hashed) IP address of the user whose API data is being updated.
// Returns:
//   error: Returns nil if the update is successful, or an error message if something goes wrong.
func UpdateAPIRelatedData(ip string) error{
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"ip": ip},
		bson.M{
			"$inc": bson.M{
				"APICalls": 1,
			},
			"$set": bson.M{
				"APILastCallDate": time.Now(),
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to update rate limit: %v", err)
	}

	return nil
}

// ResetRateLimit resets the API call count and updates the last API call timestamp for a user.
// Parameters:
//   ip (string): The anonymized (hashed) IP address of the user whose API data is being reset.
// Returns:
//   error: Returns nil if the reset is successful, or an error message if something goes wrong.
func ResetRateLimit(ip string) error{
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"ip": ip},
		bson.M{
			"$set": bson.M{
				"APICalls":        0,
				"APILastCallDate": time.Now(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %v", err)
	}

	return nil
}


// getFilesFromDir retrieves the number of files, total used space, and file metadata from a specified directory.
// Parameters:
//   DirPath (string): The path to the directory containing the users files.
// Returns:
//   filesNumber (int): The total number of files found in the directory.
//   usedSpace (float64): The total size in bytes of all files in the directory.
//   files ([]File): A list of File objects representing metadata for each file.
//   error: Returns an error if any issue occurs during processing.
func getFilesFromDir(DirPath string) (int, float64, []string, error) {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("FILES_COLLECTION"))
	godotenv.Load()

	filesArray, err := utils.GetStoredFiles(DirPath)
	var filesNumber int
	var usedSpace float64
	var ids []string

	if err != nil {
		return 0, 0.0, []string{}, fmt.Errorf("error accessing user's directory")
	}
	
	for _, arq := range filesArray {
		IdPublic := strings.Split((filepath.Base(arq)), ".")[0]
		fileNow, err := GetFileFromID(IdPublic, "public")
		if err != nil {
			return 0, 0.0, []string{}, fmt.Errorf("error retrieving file from ID %s: %v", IdPublic, err)
		}

		ids = append(ids, fileNow.IdPublic)
		filesNumber += 1
		usedSpace += fileNow.Size
	}

	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	return filesNumber, usedSpace, ids, nil
}

// GetUser retrieves the user data from the database based on the provided IP address.
// Parameters:
//   ip (string): The IP address of the user to retrieve.
// Returns:
//   User: The user data corresponding to the given IP address.
//   error: An error if the user cannot be found or if there is an issue during the query.
func GetUser(ip string) (User, error) {
	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	var user User

	filter := bson.D{{Key: "ip", Value: ip}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return User{}, fmt.Errorf("error while searching for user in database")
	}

	return user, nil
}

// DeleteUser deletes a user and all their associated files from the system and database.
// Parameters:
//   ip (string): The IP address of the user to delete.
// Returns:
//   error: An error if there is any issue during the process (deleting files, database operations, etc.).
func DeleteUser(ip string) error {
	user, err := GetUser(ip)
	if err != nil {
		return err
	}

	for _, value := range user.Files {
		file, err := GetFileFromID(value, "public")

		if err != nil {
			return err
		}

		_, err = DeleteFile(file.IdPrivate)
		if err != nil {
			return err
		}
	}
	
	path_ := filepath.Join(os.Getenv("SAVE_PATH") + ip)

	err = os.RemoveAll(path_)
	if err != nil {
		return fmt.Errorf("error deleting files from system")
	}

	ChangeCollection(os.Getenv("DB_NAME"), os.Getenv("USERS_COLLECTION"))
	filter := bson.D{{Key: "ip", Value: ip}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, filter)

	if err != nil {
		return fmt.Errorf("error while searching for user in database")
	}

	if result.DeletedCount < 1 {
		return fmt.Errorf("the file was not deleted from the database")
	}

	return nil
}


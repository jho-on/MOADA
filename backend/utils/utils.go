// Package utils provides utility functions for common operations such as email validation,
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/mail"
	"io"
	"regexp"

	"github.com/dutchcoders/go-clamd"
	"os"
	"path/filepath"
)



// ValidateEmail validates an email address with regexp.
// Parameters:
//   email (string): The email address to be validated.
// Returns:
//   bool: Returns `true` if the email is valid, and `false` otherwise.
func ValidateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}


// EncryptString hashes the input string using SHA-256 and returns the hash in hexadecimal format.
// Parameters:
//   str (string): The string to be hashed.
// Returns:
//   string: The SHA-256 hash of the input string in hexadecimal format.
func EncryptString(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}


// CheckVirus scans the given file for viruses using ClamAV.
// Parameters:
//   filepath (string): The path to the file to be scanned for viruses.
// Returns:
//   bool: Returns `true` if a virus is detected, `false` otherwise.
//   error: Returns an error if there is an issue while scanning the file, nil otherwise.
func CheckVirus(filepath string) (bool, error) {
	clam := clamd.NewClamd("/var/run/clamav/clamd.ctl")
	res, err := clam.ScanFile(filepath)

	if err != nil {
		return false, fmt.Errorf("error while checking file with antivirus ")
	}

	for r := range res {
		if r.Status == clamd.RES_FOUND {
			return true, nil // Virus
		}
	}

	return false, nil // No virus
}


// FileExists checks if a file exists at the given filepath.
// Parameters:
//   filepath (string): The path to the file or directory to check.
// Returns:
//   bool: Returns `true` if the file or directory exists, `false` otherwise.
func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return !os.IsNotExist(err)
}


// GetStoredFiles retrieves all file paths (not directories) from the specified directory.
// Parameters:
//   path (string): The directory path to search for files.
// Returns:
//   []string: A slice of file paths found in the directory.
//   error: An error if the directory does not exist or if there is an issue reading the directory.
func GetStoredFiles(path string) ([]string, error){
	
	var files []string

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return []string{}, fmt.Errorf("directory does not exist: %v", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return []string{}, fmt.Errorf("it was not possible to read the user's directory")
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			filepath := filepath.Join(path, entry.Name())
			files = append(files, filepath)
		}
	}

	return files, nil
}


// CopyFileWithoutMetadata copies a file from the input path to the output path without copying its metadata (e.g., timestamps, ownership).
// Parameters:
//   inputFilePath (string): The path to the source file.
//   outputFilePath (string): The path to the destination file.
// Returns:
//   error: An error if there was a problem opening the input file, creating the output file, or copying the contents.
func CopyFileWithoutMetadata(inputFilePath, outputFilePath string) error {
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return fmt.Errorf("error opening input file: %v", err)
	}
	
	defer inputFile.Close()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return fmt.Errorf("error copying file: %v", err)
	}

	return nil
}

// GetFolderSize walks through the specified directory and calculates the total size of all files and subdirectories inside it
// Parameters:
//   path (string): The path to the directory to be checked.
// Returns:
//   float64: The total size of the directory in bytes.
func GetFolderSize(path string) float64 {
    var dirSize float64 = 0

    readSize := func(path string, file os.FileInfo, err error) error {
        if !file.IsDir() {
            dirSize += float64(file.Size())
        }

        return nil
    }

    filepath.Walk(path, readSize)

    return float64(dirSize)
}
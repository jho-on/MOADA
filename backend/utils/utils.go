// Package utils provides utility functions for common operations such as email validation,
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/mail"
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

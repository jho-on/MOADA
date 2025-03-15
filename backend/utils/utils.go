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

func ValidateEmail(email string) bool {
	_, err := mail.ParseAddress(email)

	if err != nil {
		return false
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func EncryptString(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

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

func FileExists(filepath string) bool {

	_, err := os.Stat(filepath)

	return !os.IsNotExist(err)
}

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

package profile

import (
	"hillside/internal/utils"
	"os"
)
func getProfilePath(username string, path string) (*string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	profilePath := homeDir + "/.hillside/" +username+"_profile.json"
	if path != "" {
		profilePath = path
	}

	_, err = os.Stat(profilePath)
	if os.IsNotExist(err) {
		return nil, utils.ProfileNotFound
	}
	return &profilePath, nil
}

func createProfilePath(username string) (*string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	profilePath := homeDir + "/.hillside/" + username + "_profile.json"

	err = os.MkdirAll(homeDir+"/.hillside", os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &profilePath, nil
}

func CheckUsers() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	profileDir := homeDir + "/.hillside"
	files, err := os.ReadDir(profileDir)
	if err != nil {
		return nil, err
	}

	var users []string
	for _, file := range files {
		if !file.IsDir() && utils.IsJSONFile(file.Name()) {
			username := file.Name()[:len(file.Name())-13]
			users = append(users, username)
		}
	}
	return users, nil
}
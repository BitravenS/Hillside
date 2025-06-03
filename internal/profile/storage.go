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
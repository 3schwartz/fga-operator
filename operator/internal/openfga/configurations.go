package openfga

import (
	"fmt"
	"os"
)

const OpenfgaApiUrl = "OPEN_FGA_API_URL"
const OpenFgaApiToken = "OPEN_FGA_API_TOKEN"

type Config struct {
	ApiUrl   string
	ApiToken string
}

func NewConfig() (Config, error) {
	apiUrl, err := getEnv(OpenfgaApiUrl)
	if err != nil {
		return Config{}, err
	}
	apiToken, err := getEnv(OpenFgaApiToken)
	if err != nil {
		return Config{}, err
	}

	return Config{
		ApiUrl:   apiUrl,
		ApiToken: apiToken,
	}, nil
}

func getEnv(value string) (string, error) {
	variable := os.Getenv(value)
	if variable == "" {
		return "", fmt.Errorf("environment variable %s not found", value)
	}
	return variable, nil
}

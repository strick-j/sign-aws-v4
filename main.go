package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

func main() {
	log.Info("Initializing AWS Signer")

	creds := credentials.NewEnvCredentials()

	// Retrieve the credentials value
	credValue, err := creds.Get()
	if err != nil {
		// handle error
	}

	log.Info("Credentials: %s", credValue)

}

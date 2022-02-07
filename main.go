package main

import (
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("Initializing AWS Signer")

	sess := session.Must(session.NewSession())

	creds := stscreds.NewCredentials(sess, "myRoleArn")

	// Retrieve the credentials value
	credValue, err := creds.Get()
	if err != nil {
		log.Warn("Error retrieving credentials %s", err)
		// handle error
	}

	log.Info("Credentials: %s", credValue)

}

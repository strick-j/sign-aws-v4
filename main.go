package main

import (
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("Initializing AWS Signer")

	mySession := session.Must(session.NewSession())

	svc := ec2metadata.New(mySession)

	log.Info("MetaData Info:", svc.ClientInfo)

	creds := ec2rolecreds.NewCredentials(mySession)

	// Retrieve the credentials value
	credValue, err := creds.Get()
	if err != nil {
		log.Warn("Error retrieving credentials:", err)
		// handle error
	}

	log.Info("Credentials:", credValue)

}

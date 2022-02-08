package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
	log "github.com/sirupsen/logrus"
)

type SigHeader struct {
	Authorization     string `json:"Authorization"`
	Signature         string `json:"Signature"`
	XAmzDate          string `json:"x-amz-date"`
	XAmzSecurityToken string `json:"x-amz-security-token"`
	Host              string `json:"host"`
}

type Conjur struct {
	url    string // Conjur Host e.g. https://conjur.yourdomain.com
	acct   string // Conjur Account e.g. default
	secret string // Conjur Secret e.g. policy/path/variable-id
	host   string // Host to Authenticate as e.g. host/policy/prefix/id
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the Info severity or above.
	log.SetLevel(log.InfoLevel)
}

func main() {

	// Returns initialized Provider using EC2 Instance Metadata details
	svc := ec2rolecreds.New()

	// Retrieve retrieves credentials from the EC2 service.
	creds, err := svc.Retrieve(context.Background())
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Unable to retrieve credentials from EC2 Service: ")
	}

	// Create request
	host := "sts.amazonaws.com/"
	req, err := http.NewRequest("GET", host, nil)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Expected no error from request")
	}
	req.URL.Path = `/?Action=GetCallerIdentity&Version=2011-06-15`

	// Create Signer
	signer := v4.NewSigner()
	err = signer.SignHTTP(context.Background(), creds, req, `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`, "sts", "us-west-2", time.Unix(0, 0))
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Expected no error from request")
	}

	// Create JSON from signer response header
	sigHeader := &SigHeader{Authorization: req.Header.Get("Authorization"), XAmzDate: req.Header.Get("X-Amz-Date"), XAmzSecurityToken: req.Header.Get("X-Amz-Security-Token"), Host: host}
	headerJson, err := json.Marshal(sigHeader)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Unable to create JSON Header for Conjur Auth Request")
	} else {
		log.WithFields(log.Fields{"event": string(headerJson)}).Info("Created JSON Header for Conjur Auth Request")
	}

	// Declare / Read in Conjur Information
	conjur := Conjur{
		url:    "https://<conjur host placeholder>",
		acct:   "<conjur account placeholder>",
		secret: "<secret placehoser>",
		host:   "<host placehoder>",
	}

	// Build Conjur Auth URL
	// Authorization URL = "Conjur Host Name" + "Authenticator" + "Account" + "Escaped Host Name" + "/authenticate"
	authUrl := conjur.url + "/authn-iam/prod/" + conjur.acct + "/" + url.QueryEscape(conjur.host) + "/authenticate"

	client := &http.Client{}
	conjurReq, _ := http.NewRequest(http.MethodPost, authUrl, bytes.NewBuffer(headerJson))

	resp, err := client.Do(conjurReq)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("No response from Conjur Host")
	}
	if resp.StatusCode == 401 {
		log.WithFields(log.Fields{"Status Code": resp.StatusCode}).Fatal(ioutil.ReadAll(resp.Body))
	}

	defer resp.Body.Close()

	log.Info("Conjur Auth Response status code: ", resp.StatusCode)

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info("No response recieved from host: ", err)
	}
	log.Info("Byte Response: ", string(respBytes))
}

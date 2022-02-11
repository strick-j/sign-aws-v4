package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
	"github.com/codingconcepts/env"
	log "github.com/sirupsen/logrus"
)

type SigHeader struct {
	Host              string `json:"host"`
	XAmzDate          string `json:"x-amz-date"`
	XAmzSecurityToken string `json:"x-amz-security-token"`
	XAmzContentSHA256 string `json:"x-amz-content-sha256"`
	Authorization     string `json:"authorization"`
}

type ConjurDetails struct {
	Url       string `env:"CONJUR_APPLIANCE_URL" required="true"` // Conjur Host e.g. https://conjur.yourdomain.com
	Acct      string `env:"CONJUR_ACCOUNT" required="true"`       // Conjur Account e.g. default
	HostId    string `env:"CONJUR_AUTHN_LOGIN" required="true"`   // Host to Authenticate as e.g. host/policy/prefix/id
	ServiceId string `env:"AUTHN_IAM_SERVICE_ID" required="true"` // Authentication Service e.g. prod
}

var (
	xAmzContentSHA256 string = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	serviceID         string = "sts"       // Used in signer function
	region            string = "us-east-1" // Region must be us-east-1 for the IAM Service Call
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the Info severity or above.
	log.SetLevel(log.InfoLevel)
}

func main() {
	// Returns initialized Provider using EC2 IMDS Client by default
	svc := ec2rolecreds.New(func(options *ec2rolecreds.Options) {
		expTime := time.Now().UTC().Add(1 * time.Hour).Format("2006-01-02T15:04:05Z")
		config.WithRegion(region)
		log.WithFields(log.Fields{"Category": "Credentials"}).Debug("Credential Expiration: ", expTime)
	})

	// Retrieve retrieves credentials from the EC2 service.
	creds, err := svc.Retrieve(context.Background())
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Unable to retrieve credentials from EC2 Service")
	}
	log.WithFields(log.Fields{"Category": "Credentials"}).Debug("AccessKeyID: ", creds.AccessKeyID)
	log.WithFields(log.Fields{"Category": "Credentials"}).Debug("SecretKeyID: ", creds.SecretAccessKey)
	log.WithFields(log.Fields{"Category": "Credentials"}).Debug("Session Token: ", creds.SessionToken)

	// Create STS Request URL
	stsUrl := &url.URL{
		Scheme:   "https",
		Host:     "sts.amazonaws.com",
		Path:     "/",
		RawQuery: "Action=GetCallerIdentity&Version=2011-06-15",
	}
	stsReq, err := http.NewRequest("GET", stsUrl.String(), nil)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Expected no error from request")
	}

	// Create Signer using aws-sdk-go-v2/aws/signer
	signer := v4.NewSigner()
	err = signer.SignHTTP(context.Background(), creds, stsReq, xAmzContentSHA256, serviceID, region, time.Now().UTC(), func(o *v4.SignerOptions) {
		o.DisableURIPathEscaping = false
		o.LogSigning = true
	})
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Expected no error from request")
	}

	// Save a copy of this request for debugging.
	stsRequestDump, err := httputil.DumpRequestOut(stsReq, true)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Error dumping Conjur http.request")
	}
	log.WithFields(log.Fields{"category": "http.request"}).Debug("STS HTTP request: ", string(stsRequestDump))

	// Create JSON from signer response header
	sigHeader := SigHeader{
		Host:              stsUrl.Host,
		XAmzDate:          stsReq.Header.Get("X-Amz-Date"),
		XAmzSecurityToken: stsReq.Header.Get("X-Amz-Security-Token"),
		Authorization:     stsReq.Header.Get("Authorization"),
	}
	payload, err := json.Marshal(sigHeader)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Unable to create JSON Header for Conjur Auth Request")
	}
	log.WithFields(log.Fields{"category": "header"}).Debug("Json Header from Struct: ", string(payload))

	// Declare / Read in Conjur Information
	conjur := ConjurDetails{}
	if err := env.Set(&conjur); err != nil {
		log.WithFields(log.Fields{"Environment Variable": err}).Fatal("Required Environment Variable not set")
	}

	// Ensure Conjur Details are available and the fields aren't empty
	v := reflect.ValueOf(conjur)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Interface() == "" {
			log.Info("Conjur Environment Variable not set: ", v.Field(i).Interface())
		}
	}

	// Build Conjur URL (Path Escape required on HOST ID to convert / to %2F)
	authUrl := conjur.Url + "/authn-iam/" + conjur.ServiceId + "/" + conjur.Acct + "/" + url.PathEscape(conjur.HostId) + "/authenticate"
	log.WithFields(log.Fields{"category": "url"}).Debug("Conjur Auth URL: ", authUrl)

	// Generate Conjur Client
	client := &http.Client{}
	conjurReq, err := http.NewRequest("POST", authUrl, bytes.NewBuffer(payload))
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Expected no error from request")
	}
	conjurReq.Header.Add("Content-Type", "text/plain")
	conjurReq.Header.Add("Accept", "*/*")

	// Save a copy of this request for debugging.
	requestDump, err := httputil.DumpRequestOut(conjurReq, true)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Error dumping Conjur http.request")
	}
	log.WithFields(log.Fields{"category": "http.request"}).Debug("Conjur HTTP request: ", string(requestDump))

	resp, err := client.Do(conjurReq)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("No response from Conjur Host")
	} else if resp.StatusCode == 401 || resp.StatusCode == 404 {
		log.WithFields(log.Fields{"Status Code": resp.StatusCode}).Fatal(resp.Status)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"event": err}).Fatal("Response Body Empty")
	}
	log.WithFields(log.Fields{"Response": "Success"}).Info("Host Response: ", string(respBytes))
}

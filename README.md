# sign-aws-v4

sign-aws-v4 utilizes various aws-go-sdk-v2 packages to do the following:
1. Obtain an EC2 Host IAM Role (note: the host must have an IAM role for this to run properly)
2. Generate tempory AWS Security Credentials (AccessKeyID, SecretKeyID, and SessionToken)
3. Use the generated credentials to obtain a SigV4 Signature. A basic request is shown below:
```
GET https://sts.amazonaws.com/?Action=GetCallerIdentity&Version=2011-06-15 HTTP/1.1
Content-Type: application/x-www-form-urlencoded; charset=utf-8
Host: sts.amazonaws.com
X-Amz-Date: 20150830T123600Z
```
4. Uses the signature as part of the body in a request to CyberArk Conjur for authentication based on the EC2 host IAM Role

### EC2 Usage
Prior to using the application, ensure the following Environment Variables are set and available:
```
$ export CONJUR_APPLIANCE_URL=https://conjur.yourorg.com
$ export AUTHN_IAM_SERVICE_ID=<service-id>
$ export CONJUR_AUTHN_LOGIN=host/cust-portal/<aws-account-id>/<iam-role-name>
$ export CONJUR_ACCOUNT=<account>
```
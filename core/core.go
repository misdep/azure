package core

import(
  "time"
  "net/url"
  "net/http"
  "fmt"
  "log"
  "bytes"
  "strings"
  "encoding/base64"
  "crypto/hmac"
  "crypto/sha256"
  "sort"
  "strconv"
  "io"
)

const ms_date_layout = "Mon, 2 Jan 2006 15:04:05 GMT"
const version = "2009-09-19"

type Credentials struct {
  Account string
  AccessKey string
}

type AzureRequest struct {
  Method string
  Container string
  Resource string
  RequestTime time.Time
  Request *http.Request
  Header map[string]string
  Body io.Reader
}

type Core struct {
  Credentials Credentials
  AzureRequest AzureRequest
}

func New(credentials Credentials, azureRequest AzureRequest) *Core {
  return &Core{
    Credentials: credentials,
    AzureRequest: azureRequest}
}

func (core Core) addCustomInformationsToHeader() {
  for key, value := range core.AzureRequest.Header {
    core.AzureRequest.Request.Header.Add(key, value)
  }
}

func (core Core) PrepareRequest() *http.Request {
  req, err := http.NewRequest(strings.ToUpper(core.AzureRequest.Method), core.RequestUrl(), core.AzureRequest.Body)

  if err != nil {
    log.Fatal(err)
  }

  core.AzureRequest.Request = req
  core.addCustomInformationsToHeader()
  core.complementHeaderInformations()

  return req
}

func (core Core) RequestUrl() string {
  return fmt.Sprintf("%s%s%s", core.webService(), core.AzureRequest.Container, core.AzureRequest.Resource)
}

func (core Core) complementHeaderInformations() {
  core.AzureRequest.Request.Header.Add("x-ms-date", core.formattedRequestTime())
  core.AzureRequest.Request.Header.Add("x-ms-version", version)
  core.AzureRequest.Request.Header.Add("Authorization", core.authorizationHeader())
}

func (core Core) authorizationHeader() string {
  return fmt.Sprintf("SharedKey %s:%s", core.Credentials.Account, core.signature())
}

/*
Based on Azure docs:
  Link: http://msdn.microsoft.com/en-us/library/windowsazure/dd179428.aspx#Constructing_Element

  1) Retrieve all headers for the resource that begin with x-ms-, including the x-ms-date header.
  2) Convert each HTTP header name to lowercase.
  3) Sort the headers lexicographically by header name, in ascending order. Note that each header may appear only once in the string.
  4) Unfold the string by replacing any breaking white space with a single space.
  5) Trim any white space around the colon in the header.
  6) Finally, append a new line character to each canonicalized header in the resulting list. Construct the CanonicalizedHeaders string by concatenating all headers in this list into a single string.
*/
func (core Core) canonicalizedHeaders() string {
  var buffer bytes.Buffer

  for key, value := range core.AzureRequest.Request.Header {
    lowerKey := strings.ToLower(key)

    if strings.HasPrefix(lowerKey, "x-ms-") {
      if buffer.Len() == 0 {
        buffer.WriteString(fmt.Sprintf("%s:%s", lowerKey, value[0]))
      }else {
        buffer.WriteString(fmt.Sprintf("\n%s:%s", lowerKey, value[0]))
      }
    }
  }

  splitted := strings.Split(buffer.String(), "\n")
  sort.Strings(splitted)

  return strings.Join(splitted, "\n")
}

func (core Core) canonicalizedResource() string {
  var buffer bytes.Buffer

  u, err := url.Parse(core.RequestUrl())

  if err != nil {
    log.Fatal(err)
  }

  buffer.WriteString(fmt.Sprintf("/%s/%s", core.Credentials.Account, core.AzureRequest.Container))
  queries := u.Query()

  for key, values := range queries {
    sort.Strings(values)
    buffer.WriteString(fmt.Sprintf("\n%s:%s", key, strings.Join(values, ",")))
  }

  return buffer.String()
}

func (core Core) contentLength() (contentLength string) {
  if core.AzureRequest.Request.Method == "PUT" {
    contentLength = strconv.FormatInt(core.AzureRequest.Request.ContentLength, 10)
  }

  return
}

func (core Core) formattedRequestTime() string {
  return core.AzureRequest.RequestTime.Format(ms_date_layout)
}

/*
params:
 HTTP Verb
 Content-Encoding
 Content-Language
 Content-Length
 Content-MD5
 Content-Type
 Date
 If-Modified-Since
 If-Match
 If-None-Match
 If-Unmodified-Since
 Range
*/
func (core Core) signature() string {
  signature := fmt.Sprintf("%s\n\n\n%s\n\n\n\n\n\n\n\n\n%s\n%s",
    strings.ToUpper(core.AzureRequest.Method),
    core.contentLength(),
    core.canonicalizedHeaders(),
    core.canonicalizedResource())

  decodedKey, _ := base64.StdEncoding.DecodeString(core.Credentials.AccessKey)

  sha256 := hmac.New(sha256.New, []byte(decodedKey))
  sha256.Write([]byte(signature))

  return base64.StdEncoding.EncodeToString(sha256.Sum(nil))
}

func (core Core) webService() string {
  return fmt.Sprintf("https://%s.blob.core.windows.net/", core.Credentials.Account)
}

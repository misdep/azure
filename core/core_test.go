package core

import(
  "fmt"
  "testing"
  "time"
  "net/http"
  "net/http/httptest"
  . "launchpad.net/gocheck"
)

func Test(t *testing.T) {
  TestingT(t)
}

var _ = Suite(&CoreSuite{})

// Global
var requestTime = time.Date(2013, time.November, 02, 15, 0, 0, 0, time.UTC)
var credentials = Credentials{
  Account: "sampleAccount",
  AccessKey: "secretKey"}

var azureRequest = AzureRequest{
  Method: "put",
  Container: "samplecontainer",
  Resource: "?restype=container",
  RequestTime: requestTime}

type CoreSuite struct{
  core *Core
}

func (s *CoreSuite) SetUpSuite (c *C) {
  s.core = New(credentials, azureRequest)
}

func (s *CoreSuite) Test_RequestUrl(c *C) {
  expected := "https://sampleAccount.blob.core.windows.net/samplecontainer?restype=container"
  c.Assert(s.core.RequestUrl(), Equals, expected)
}

func (s *CoreSuite) Test_Request(c *C) {
  handle := func(w http.ResponseWriter, r *http.Request) {
    c.Assert(r.URL.Scheme, Equals, "https")
    c.Assert(r.URL.Host, Equals, "sampleAccount.blob.core.windows.net")
    c.Assert(r.URL.Path, Equals, "/samplecontainer")

    //METHOD
    c.Assert(r.Method, Equals, "PUT")
    // HEADER
    c.Assert(r.Header.Get("x-ms-date"), Equals, "Sat, 02 Nov 2013 15:00:00 GMT")
    c.Assert(r.Header.Get("x-ms-version"), Equals, "2009-09-19")
    c.Assert(r.Header.Get("Authorization"), Equals, "SharedKey sampleAccount:h0VRxbQipkWe0762ni41UQrKqV5h/j5gMlJDjb0tvys=")
  }

  req := s.core.PrepareRequest()
  w := httptest.NewRecorder()

  handle(w, req)
}

func (s *CoreSuite) Test_RequestWithCustomHeaders(c *C) {
  handle := func(w http.ResponseWriter, r *http.Request) {
    // HEADER
    c.Assert(r.Header.Get("some"), Equals, "header key")
    c.Assert(r.Header.Get("x-ms-blob-type"), Equals, "BlockBlob")
    c.Assert(r.Header.Get("x-ms-date"), Equals, "Sat, 02 Nov 2013 15:00:00 GMT")
    c.Assert(r.Header.Get("x-ms-version"), Equals, "2009-09-19")
    c.Assert(r.Header.Get("Authorization"), Equals, "SharedKey sampleAccount:BXo6wDPzH6TAUVgg0immVsr/1x6xlBLC3/8W71iRMmo=")
  }

  s.core.AzureRequest.Header = map[string]string{
    "x-ms-blob-type":"BlockBlob",
    "some": "header key"}

  req := s.core.PrepareRequest()
  w := httptest.NewRecorder()

  handle(w, req)
}

func (s *CoreSuite) Test_CanonicalizedHeaders(c *C) {
  req, err := http.NewRequest("GET", "http://example.com", nil)

  if err != nil {
    c.Error(err)
  }

  req.Header.Add("nothing", "important")
  req.Header.Add("X-Ms-Version", "2009-09-19")
  req.Header.Add("X-Ms-Date", "Fri, 22 Nov 2013 15:00:00 GMT")
  req.Header.Add("X-Ms-Blob-Type", "BlockBlob")
  req.Header.Add("Content-Type", "text/plain; charset=UTF-8")

  s.core.AzureRequest.Request = req

  expected := fmt.Sprintf("x-ms-blob-type:%s\nx-ms-date:%s\nx-ms-version:%s", "BlockBlob", "Fri, 22 Nov 2013 15:00:00 GMT", "2009-09-19")
  c.Assert(s.core.canonicalizedHeaders(), Equals, expected)
}

func (s *CoreSuite) Test_CanonicalizedResource(c *C) {
  expected := "/sampleAccount/samplecontainer\nrestype:container"
  c.Assert(s.core.canonicalizedResource(), Equals, expected)
}

func (s *CoreSuite) Test_CanonicalizedResourceWithCustomParams(c *C) {
  a := AzureRequest{
  Container: "samplecontainer",
  Resource: "?restype=container&comp=list"}

  customCore := New(credentials, a)

  expected := "/sampleAccount/samplecontainer\ncomp:list\nrestype:container"
  c.Assert(customCore.canonicalizedResource(), Equals, expected)
}
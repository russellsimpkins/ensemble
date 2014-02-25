package ensemble

import (
	"encoding/json"
	"testing"
)

func TestIsValidHTTPMethod(t *testing.T) {
	var (
		pass string
		fail string
	)
	pass = "post"
	fail = "posts"

	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	worked := IsValidHTTPMethod(&pass)
	if !worked {
		t.Error("passing test failed.")
	}

	worked = IsValidHTTPMethod(&fail)
	if worked {
		t.Error("failing test passed. Check your logic")
	}

}

func TestMakeJsonData(t *testing.T) {
	var (
		api *ApiWrapper
		req []Request
		item []string
	)

	api = &ApiWrapper{
	}
	item = make([]string, 1)
	req = make([]Request, 2)
	req[0].Id = "1"
	req[0].URL = "http://localhost/test1.php"
	req[0].Method = "GET"
	req[0].Payload = "foo=bar"
	req[0].Headers = make(map[string][]string, 1)
	item[0]  = "application/x-www-form-urlencoded"
	req[0].Headers["Content-Type"] = item
	req[1].Id = "2"
	req[1].URL = "http://localhost/test2.php"
	req[1].Method = "GET"
	req[1].Payload = "foo=bar"
	req[1].Headers = make(map[string][]string, 1)
	req[1].Headers["Content-Type"] = item
	req[1].Depends = make([]Dependency, 1)
	api.Requests = req
	d, _ := json.Marshal(api)
	t.Logf("%s\n", string(d))
	t.Logf("%m\n", api)

}

/*
// This tests against a local server running php - maybe move to mocks or
// one time go http server?
func TestMakeRequest(t *testing.T) {

	var (
		url string
		method string
		data string
		response string
		responseCode int
		err error
	)

	url = "http://localhost/test.php"
	method = "POST"
	data = "{\"foo\":\"bar\"}"
	t.Logf("I'm using %s method to this url %s", method, url)
	response, responseCode, err = MakeRequest(&url,&method,nil,&data)

	if responseCode != 200 {
		t.Errorf("expected 200 but got %d",responseCode)
	}

	if err != nil {
		t.Error(err)
	}

	t.Logf("it all worked %s",response)
}
*/

package ensemble

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestIsValidHTTPMethod(t *testing.T) {
	var (
		pass string
		fail string
	)
	pass = "POST"
	fail = "POSTS"

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

func TestFormatResults(t *testing.T) {
	var testdata []string
	testdata = make([]string, 3)
	testdata[0] = `{"foo":"bar","blip":24}`
	testdata[1] = `{"bar":"foo","blip":25}`
	testdata[2] = `{"boo":"fbar","blip":26}`
	expected := `[{"blip":24,"foo":"bar"},{"bar":"foo","blip":25},{"blip":26,"boo":"fbar"}]`
	data, err := createJsonArray(testdata)
	if err != nil {
		t.Error(err)
	} else {
		if expected != data {
			t.Error("data not formatted as expected.")
		}
		t.Logf("data is: %s", data)
	}
}

func Provide1(writer http.ResponseWriter, req *http.Request) {
	writer.Write([]byte("{\"is this\":\"magic?\"}"))
}

func Provide2(writer http.ResponseWriter, req *http.Request) {
	writer.Write([]byte("{\"yup\":\"magic happens\"}"))
}

func ATest1(writer http.ResponseWriter, req *http.Request) {
	writer.Write([]byte("This worked"))
	return
}

func ATest2(writer http.ResponseWriter, req *http.Request) {
	body, _ := ioutil.ReadAll(req.Body)
	writer.Write(body)
	return
}

func StartListener() {
	http.HandleFunc("/test", Handle)
	http.HandleFunc("/test1", ATest1)
	http.HandleFunc("/test2", ATest2)
	http.HandleFunc("/provide1", Provide1)
	http.HandleFunc("/provide2", Provide2)
	srv := &http.Server{
		Handler:        nil,
		Addr:           ":8080",
		WriteTimeout:   5 * time.Second,
		ReadTimeout:    5 * time.Second,
		MaxHeaderBytes: 32768,
	}

	err := srv.ListenAndServe()
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}
}

func DontTestCallHandle(t *testing.T) {
	var (
		url          string
		method       string
		data         string
		response     string
		responseCode int
		err          error
		contentType  string
		header       *http.Header
	)
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	go StartListener()
	//url = "http://localhost/test.php?a=b"
	url = "http://localhost:8080/test"
	method = "POST"
	data = "{\"foo\":\"bar\"}"
	//data = "foo=bar&cool=false"
	//data = ""

	data = `{
    "requests": [{
        "id": "1",
        "url": "http://localhost:8080/test1",
        "method": "GET",
        "rawData": "foo=bar",
        "headers": {
            "Content-Type": ["application/x-www-form-urlencoded"]
        },
        "dependency": null
    }, {
        "id": "2",
        "url": "http://localhost:8080/test2",
        "method": "POST",
        "data": "{\"data\":[%s]}",
        "evalJson": true,
        "headers": {
            "Content-Type": ["application/x-www-form-urlencoded"]
        },
        "dependency": [{
            "request": {
                "id": "21",
                "url": "http://localhost:8080/provide1",
                "method": "GET"
            }
        }, {
            "request": {
                "id": "22",
                "url": "http://localhost:8080/provide2",
                "method": "GET"
            }
        }],
        "useData": true,
        "doJoin": true,
        "joinChar": ","
    }],
    "strictorder": true,
    "hasDependencies": true
}`

	//	data = `{"requests":[{"id":"1","url":"http://www.nytimes.com/adx/bin/adxrun.html?v=3&page=homepage.nytimes.com/index.html&positions=Box1&keywords=adxeng-adxcon-test1"},{"id":"2","url":"http://platforms.nytimes.com/mobile/v2/json/iphone/latestfeed.json","graphics8Access":true,"graphicsPath":"/mobile/v2/json/iphone/"}],"strictorder":false,"hasDependencies":false}`

	contentType = "application/json"
	// the key is to set the correct Content-Type for Go to work right!
	// so, we need to define the callers and check the content type
	contentType = "application/x-www-form-urlencoded"
	//contentType = ""
	header = &http.Header{}

	header.Add("Content-Type", contentType)
	//header = nil
	t.Logf("I'm using %s method to this url %s\n", method, url)

	req := Request{URL: url, Method: method, Header: *header, Data: data}
	res := new(Response)

	err = MakeRequest(&req, res)
	if err != nil {
		fmt.Println()
		fmt.Printf("Code: %v. Response: %s\n", responseCode, response)
		t.Error("Error: ", err, "Code: %v. Response: %s\n", responseCode, response)
	}

	t.Logf("It worked! Response data: %s", res)
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

package ensemble

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func Wrap(writer http.ResponseWriter, req *http.Request) {
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println("We got:", string(bytes))
	return
}

func ProvideJson(writer http.ResponseWriter, req *http.Request) {
	msg := `{"age":25,"weight":12}`
	writer.Write([]byte(msg))
}
func ProvideJsonAge(writer http.ResponseWriter, req *http.Request) {
	msg := `{"age":25}`
	writer.Write([]byte(msg))
}
func ProvideJsonHeight(writer http.ResponseWriter, req *http.Request) {
	msg := `{"height":"72"}`
	writer.Write([]byte(msg))
}

type Height struct {
	Height int `json:height`
}
type HeightStr struct {
	Height string `json:height`
}

func ProvideJsonHeightString(writer http.ResponseWriter, req *http.Request) {
	var (
		heighti Height
		height  int
		result  HeightStr
		hstr    string
	)
	msg, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(msg, &heighti)
	height = int(heighti.Height)
	hstr = fmt.Sprintf("\"%i' %i\"", (height / 12), (height % 12))
	result = HeightStr{}
	result.Height = hstr
	b, _ := json.Marshal(result)
	writer.Write(b)
}

func StartListening() {
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

func TestCallHandle(t *testing.T) {

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
	go StartListening()
	//url = "http://localhost/test.php?a=b"
	url = "http://localhost:8080/test"
	method = "post"
	data = "{\"foo\":\"bar\"}"
	//data = "foo=bar&cool=false"
	//data = ""
	data = `{"requests":[{"id":"1","url":"http://localhost:8080/test1","method":"GET","rawData":"foo=bar","headers":{"Content-Type":["application/x-www-form-urlencoded"]},"dependency":null},{"id":"2","url":"http://localhost:8080/test2","method":"POST","payload":"{\"data\":[%s]}", "evalJson":true,"headers":{"Content-Type":["application/x-www-form-urlencoded"]},"dependency":[{"request":{"id":"21","url":"http://localhost:8080/provide1","method":"GET"}},{"request":{"id":"22","url":"http://localhost:8080/provide2","method":"GET"}}],"useData":true,"doJoin":true,"joinChar":","}],"strictorder":true,"hasDependencies":true}`

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
	//
	t.Logf("%s", response)
}

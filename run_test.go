package ensemble
import (
	"fmt"
	"net/http"
	"io/ioutil"
	"time"
	"testing"
)

func Wrap(writer http.ResponseWriter, req *http.Request) {
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println("We got:", string(bytes))
	return
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
	body,_ := ioutil.ReadAll(req.Body)
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

func TestCallHandle(t *testing.T) {

	var (
		url string
		method string
		data string
		response string
		responseCode int
		err error
		contentType string
		header *http.Header

	)
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	go StartListener()
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

	header.Add("Content-Type",contentType)
	//header = nil
	//fmt.Printf("I'm using %s method to this url %s\n", method, url)
	response, responseCode, _, err = MakeRequest(&url,&method,header,&data)
	if err != nil {
		fmt.Println()
		fmt.Printf("Code: %v. Response: %s\n", responseCode,response)
		t.Error("Error: ", err, "Code: %v. Response: %s\n", responseCode,response)
	}
	//
	t.Logf("%s",response)
}

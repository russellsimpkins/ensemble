package ensemble

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const DefaultTimeout = 10 * time.Second

func HowTo() (msg string) {
	msg = `This service orchestrates RestFUL API calls. You come up with a set of RestFUL API calls
 you want to make, put them into JSON encoding according to our format and you call this. Here is the most basic example.

{
    "requests": [{
        "id": "1",
        "url": "http://localhost:8080/test1?foo=bar",
        "method": "GET"
    }, {
        "id": "2",
        "url": "http://localhost:8080/test2",
        "method": "POST",
        "payload": "boo=far"
    }],
    "strictorder": false
}

This will call either call each request, but is not guaranteed to be in the order you specify. Order is done using the json.
The id is there to identify each request. You can make the call follow the order by setting "strictorder":true e.g.

{
    "requests": [{
        "id": "1",
        "url": "http://localhost:8080/test1?foo=bar",
        "method": "GET"
    }, {
        "id": "2",
        "url": "http://localhost:8080/test2",
        "method": "POST",
        "payload": "boo=far"
    }],
    "strictorder": false
}

You can also add dependency calls into the mix. Let's say you want an API call done before you make your call to call id=2

{
    "requests": [{
        "id": "1",
        "url": "http://localhost:8080/test1",
        "method": "GET",
        "dependency": null
    }, {
        "id": "2",
        "url": "http://localhost:8080/test2",
        "method": "POST",
        "payload": "%s",
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
    "strictorder": true
}`
	return
}

// utility method to validate the HTTP method someone desires to use.
func IsValidHTTPMethod(method *string) (valid bool) {
	var validMethod string
	validMethod = "^(GET|PUT|POST|DELETE)$" // we only support these
	*method = strings.ToUpper(*method)      // make sure it's upper case
	valid, _ = regexp.Match(validMethod, []byte(*method))
	return
}

/*
 * we expect url to start with http or https. GET requests should have ?name=val.
 * data will always go in the body of the request. It's more for backward compatability
 * since I know of services that combine request data with query strings.
 */
func MakeRequest(url *string, method *string, header *http.Header, rawData *string) (result string, code int, hdr http.Header, err error) {

	var (
		empty   string
		body    []byte
		client  *http.Client
		request *http.Request
		resp    *http.Response
		sr      io.Reader
	)

	*method = strings.ToUpper(*method)
	if !IsValidHTTPMethod(method) {
		result = "Invalid HTTP Method requested"
		code = 500
		err = errors.New(result)
		return
	}

	if len(*rawData) > 0 {
		sr = strings.NewReader(*rawData)
	}

	request, err = http.NewRequest(*method, *url, sr)

	if header != nil {
		request.Header = *header
	}

	// set a default content type
	t := request.Header.Get("Content-Type")
	if len(t) <= 0 {
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	client = &http.Client{
		CheckRedirect: nil,
	}

	resp, err = client.Do(request)

	if err != nil {
		return empty, 500, nil, err
	}
	hdr = resp.Header
	code = resp.StatusCode
	body, err = ioutil.ReadAll(resp.Body)
	result = string(body)
	return
}

// utility to replace header values
func replaceHeader(req *Request, header *http.Header) {
	// replace headers with request over-rides if they exist
	for name, vals := range req.Headers {
		header.Del(name)
		for _, val := range vals {
			header.Add(name, val)
		}
	}
}

// process request dependencies
func ProcessDependencies(req *Request, resp *Response, header *http.Header) {
	var (
		err        error
		respHeader http.Header
	)

	deps := req.Depends
	results := make([]string, len(deps))
	for index, dep := range deps {
		dep := dep
		code := 0
		results[index], code, respHeader, err = MakeRequest(&dep.Request.URL, &dep.Request.Method, header, &dep.Request.Payload)
		if code != 200 || err != nil {
			resp.Id = dep.Request.Id
			resp.Code = code
			resp.Payload = err.Error()
			return
		}
	}

	if req.UseData {
		if req.DoJoin {
			s := fmt.Sprintf(req.Payload, strings.Join(results, req.JoinChar))
			req.Payload = s

		}
	}

	if req.UseDepHeader && len(req.DepHeader) > 0 {
		for _, name := range req.DepHeader {
			header.Del(name)
			header.Add(name, respHeader.Get(name))
		}
	}
}

// for making async requests
func AsyncRequest(req *Request, resp *Response, header *http.Header, c chan int) {

	var (
		err          error
		hd           http.Header
		returnObject interface{}
	)

	hd = *header
	if req.Headers != nil {
		replaceHeader(req, &hd)
	}

	if req.Depends != nil {
		resp.Code = 200
		ProcessDependencies(req, resp, &hd)
		if resp.Code != 200 {
			log.Println("Failed processing dependencies.")
			return
		}
	}
	resp.Id = req.Id
	resp.Code = 404
	resp.Payload = `{"reason":"Request timed out"}`
	resp.Payload, resp.Code, _, err = MakeRequest(&req.URL, &req.Method, &hd, &req.Payload)
	if err != nil {
		msg := `{"reason":"%s"}`
		msg = fmt.Sprintf(msg, err.Error)
		resp.Payload = msg
	} else {
		if req.EvalRespJson {
			returnObject = new(interface{})
			err = json.Unmarshal([]byte(resp.Payload), returnObject)
			if err != nil {
				msg := `{"reason":"%s"}`
				msg = fmt.Sprintf(msg, err.Error())
				json.Unmarshal([]byte(msg), returnObject)
				fmt.Println("", err)
			}
			resp.Object = returnObject
		}
	}
	c <- 0

}

// for maying syncronous requests
func SyncRequest(req *Request, resp *Response, header *http.Header) {
	var (
		err          error
		hd           http.Header
		returnObject interface{}
	)

	hd = *header
	if req.Headers != nil {
		replaceHeader(req, &hd)
	}

	replaceHeader(req, &hd)
	if req.Depends != nil {
		resp.Code = 200
		ProcessDependencies(req, resp, &hd)
		if resp.Code != 200 {
			fmt.Println("fail")
			return
		}
	}
	resp.Id = req.Id
	resp.Payload, resp.Code, _, err = MakeRequest(&req.URL, &req.Method, &hd, &req.Payload)
	if err != nil {
		resp.Payload = err.Error()
	} else {
		if req.EvalRespJson {
			returnObject = new(interface{})
			err = json.Unmarshal([]byte(resp.Payload), returnObject)
			if err != nil {
				fmt.Println("Doh: ", err)
			}
			resp.Object = returnObject
		}
	}
}

func ProcessJson(job *ApiWrapper, header *http.Header) (res Result, err error) {

	c := make(chan int, len(job.Requests))
	res.Responses = make([]Response, len(job.Requests))

	for index, _ := range job.Requests {
		if job.SyncOrder {
			SyncRequest(&job.Requests[index], &res.Responses[index], header)
			if res.Responses[index].Code != 200 {
				msg := `{"reason":"%s"}`
				msg = fmt.Sprintf(msg, res.Responses[index].Payload)
				err = errors.New(msg)
				return
			}
		} else {
			go AsyncRequest(&job.Requests[index], &res.Responses[index], header, c)
		}
	}

	// if async, wait for our response or timeout
	if !job.SyncOrder {

		// create a timeout so we don't wait forever
		var timeout <-chan time.Time
		if job.Timeout > 0 {
			timeout = time.After(time.Duration(job.Timeout))
		} else {
			timeout = time.After(DefaultTimeout)
		}

		for index := 0; index < len(job.Requests); index++ {
			select {
			case <-c:
			case <-timeout:
				log.Print("We timed out")
				break
			}
		}

	}

	return
}

/**
 * Basic handle function is called on each http request.
 */
func Handle(writer http.ResponseWriter, req *http.Request) {

	var (
		job  ApiWrapper
		err  error
		res  Result
		body []byte
	)

	body, err = ioutil.ReadAll(req.Body)

	if err != nil {
		str := fmt.Sprintf("Unable to read in the body of the request: %s", body)
		log.Print(str)
		http.Error(writer, str, 500)
		return
	}

	err = json.Unmarshal(body, &job)

	if err != nil {
		str := fmt.Sprintf("[ERROR] Unable to parse JSON: %s", err)
		http.Error(writer, str, 500)
		return
	}

	res, err = ProcessJson(&job, &req.Header)

	if err != nil {
		str := fmt.Sprintf("[ERROR] Unable to process request(s): %s", err)
		http.Error(writer, str, 500)
		return
	}

	body, _ = json.Marshal(res)
	writer.Write(body)
}

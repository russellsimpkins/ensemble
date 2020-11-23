package ensemble

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	// go-kit imports
	"github.com/go-kit/kit/endpoint"
)

const DefaultTimeout = 10 * time.Second

// MakeMagicEndpoint creates go-kit endpoint function
func MakeMagicEndpoint(magic *Magic) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(Workload)
		result, err := magic.DoMagic(req)
		return result, err
	}
}

// allows us to get the header of the original request
func (workload *Workload) SetHeader(h http.Header) {
	workload.header = h
}

func (magic *Magic) DoMagic(workload Workload) (result Result, err error) {
	err = process(workload, &result)
	return
}

// process is called from the go-kit func
// process looks at the workload and calls requests syncronously or asyncronously
func process(workload Workload, result *Result) (err error) {

	c := make(chan int, len(workload.Requests))
	result.Responses = make([]Response, len(workload.Requests))

	for index, _ := range workload.Requests {

		if workload.UseHeaders {
			replaceHeaderValues(&workload.Requests[index].Header, &workload.header)
		}

		if workload.StrictOrder {
			syncRequest(&workload.Requests[index], &result.Responses[index])
		} else {
			go asyncRequest(&workload.Requests[index], &result.Responses[index], c)
		}
	}

	// if async, wait for our responses or timeout
	if !workload.StrictOrder {

		// create a timeout so we don't wait forever
		var timeout <-chan time.Time

		if workload.Timeout > 0 {
			timeout = time.After(time.Duration(workload.Timeout))
		} else {
			timeout = time.After(DefaultTimeout)
		}

		for index := 0; index < len(workload.Requests); index++ {
			select {
			case <-c:
			case <-timeout:
				log.Warn("[process] Timed out waiting for all go routines to complete")
				break
			}
		}
	}
	return
}

// for making async requests
func asyncRequest(request *Request, response *Response, c chan int) {
	syncRequest(request, response)
	c <- 1
}

// SyncRequest will process any request dependencies and then call MakeRequest
// if there are errors, it's returned in the response
func syncRequest(request *Request, response *Response) {
	var (
		err error
	)

	if request.Dependents != nil {
		log.Debugf("[syncRequest] There are dependencies")
		processDependencies(request, response)
		if response.Code != 200 {
			log.Debugf("[syncRequest] bad response code")
			log.Debugf("[syncRequest] %#v", response)
			return
		}
	}

	log.WithFields(log.Fields{"method": request.Method, "URL": request.URL, "data": request.Data}).Debugf("[syncRequest] Making a request.")

	if err = MakeRequest(request, response); err != nil {
		log.WithFields(log.Fields{"err": err}).Error("[syncRequest] unable to call MakeRequest")
		response.Data = err.Error()
	}

	return
}

// this function handles a Request's dependencies. Rather than return an error, if there
// is a problem, the problem is in the Resposne.
// TODO - add support for "allowed response codes" t
// TODO - add support for "abort on failure = true/false" - current behavior = true
func processDependencies(request *Request, response *Response) {

	var (
		err     error
		results []Response
		dataset []string
	)

	response.Code = 200

	// sanity check
	if len(request.Dependents) == 0 {
		return
	}

	// This block of code is going to execute dependencies and stop if one has an error.
	results = make([]Response, len(request.Dependents))
	dataset = make([]string, len(request.Dependents))

	for index, dep := range request.Dependents {
		dep := dep
		err = MakeRequest(&dep.Request, &results[index])
		// if any of the caller's dependency calls fails, we fail fast
		if results[index].Code >= 200 && results[index].Code < 300 || err != nil {
			log.WithFields(log.Fields{"code": results[index].Code}).Debugf("[ProcessDependencies] bad response")
			response.Id = dep.Request.Id
			response.Code = results[index].Code
			response.Data = results[index].Data
			return
		}
		dataset[index] = results[index].Data
		if request.UseDepHeader {
			replaceHeaderValues(&request.Header, &results[index].Header)
		}
	}

	// check if the current request needs to use the data from the dependent calls
	// if so, we will set the parent's Data to the combined value of the dependent
	// call response data
	if request.UseData {
		log.Debug("[ProcessDependencies] decided to use data")
		if request.Data == "" {
			log.Error("[ProcessDependencies] missing data and UseData was set to true")
			response.Code = 500
			response.Data = "useData set to true, but data value not set"
			return
		}
		// if DoJoin, the end users wants separated results
		if request.DoJoin {
			log.Debugf("[ProcessDependencies]doing a join of the data. req.Data is %s ::", request.Data)
			s := fmt.Sprintf(request.Data, strings.Join(dataset, request.JoinChar))
			request.Data = s
			log.Debugf("[ProcessDependencies] req data is now %s", s)
		} else {
			log.Debugf("[ProcessDependencies] creating an array of the results")
			request.Data, err = createJsonArray(dataset)
		}
		log.Debugf("[ProcessDependencies] req.Data is now [[%s]]", request.Data)
	} else {
		log.Debug("[ProcessDependencies] req.UseData is false")
	}

	return
}

/*
 * we expect a valid url. GET requests support name value pairs.
 * Data will always go in the body of the request. It's more for backward compatability
 * since I know of services that combine request data with query strings.
 */
func MakeRequest(req *Request, response *Response) (err error) {

	var (
		body    []byte
		client  *http.Client
		request *http.Request
		resp    *http.Response
		sr      io.Reader
	)

	method := strings.ToUpper(req.Method)
	response.Id = req.Id

	if !IsValidHTTPMethod(&method) {
		response.Data = "Invalid HTTP Method requested."
		response.Code = 500
		return
	}

	timeout := time.Duration(1 * time.Second) // one second timeout

	if len(req.Data) > 0 {
		sr = strings.NewReader(req.Data)
	}

	if request, err = http.NewRequest(method, req.URL, sr); err != nil {
		log.WithFields(log.Fields{"method": method, "url": req.URL, "data": req.Data}).Debugf("[MakeRequest] Unable to create http.Request")
		return
	}

	request.Close = true

	if req.Header != nil {
		request.Header = req.Header
	} else if request.Header == nil {
		request.Header = make(map[string][]string)
	}

	// set a default content type if it isn't already set
	if t := request.Header.Get("Content-Type"); len(t) <= 0 {
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	client = &http.Client{
		CheckRedirect: nil,
		Timeout:       timeout,
	}

	if resp, err = client.Do(request); err != nil {
		return
	}

	defer resp.Body.Close()
	response.Header = resp.Header
	response.Code = resp.StatusCode

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		response.Data = err.Error()
	} else {
		response.Data = string(body)
	}

	log.WithFields(log.Fields{"body": string(body)}).Debugf("[MakeRequest] response returned.")
	return
}

// utility method to validate the HTTP method someone desires to use.
func IsValidHTTPMethod(method *string) (valid bool) {
	valid, _ = regexp.Match("^(GET|PUT|POST|DELETE)$", []byte(*method))
	return valid
}

// logic to replace header values in the target with those in the source
// this leaves any header values in the target that aren't in the source
// if you wanted that, don't call this fuction.
func replaceHeaderValues(target *http.Header, source *http.Header) {
	if target == nil {
		target = new(http.Header)
	}
	for name, vals := range *source {
		target.Del(name)
		for _, val := range vals {
			target.Add(name, val)
		}
	}
	return
}

// take an array of string input, which is assumed to be json encoded objects,
// and put them into a json array of, of well, whatever the encoded data is.
func createJsonArray(dataset []string) (data string, err error) {

	var (
		results []interface{}
		set     []byte
		value   interface{}
	)

	results = make([]interface{}, len(dataset))

	for index, item := range dataset {
		set = []byte(item)
		err = json.Unmarshal(set, &value)
		if err != nil {
			// unable to convert json to interface.... doh
			data = ""
			return
		}
		results[index] = value
	}

	// now that we've got an array of results, marshal back to string
	if set, err = json.Marshal(results); err != nil {
		data = ""
		return
	} else {
		data = string(set)
	}
	return data, err
}

// callers may give us a Timeout - if they do, lets divide the timeout
// by the number of requests and set each subsequent request to timeout
// at that value or we will use a default, sane timeout of 2 seconds
func calculateTimeout(show *Workload) (timeout int64) {

	timeout = 2
	if show.Timeout > 0 {
		timeout = show.Timeout
	}

	if len(show.Requests) > 1 {
		timeout = timeout / int64(len(show.Requests))
		if timeout < 1 {
			timeout = 1
		}
	}

	return timeout
}

func Help() (msg string) {
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
		"data": "boo=far"
	}],
	"strictorder": false
}

This will call each request, but is not guaranteed to be in the order you specify. Order is done using the json. The id is there to identify each request. You can make the call follow the order by setting "strictorder":true e.g.

{
	"requests": [{
		"id": "1",
		"url": "http://localhost:8080/test1?foo=bar",
		"method": "GET"
	}, {
		"id": "2",
		"url": "http://localhost:8080/test2",
		"method": "POST",
		"data": "boo=far"
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
		"data": "%s",
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

// Handle function is entry point for all http request.
func Handle(writer http.ResponseWriter, req *http.Request) {

	var (
		work Workload
		err  error
		res  Result
		body []byte
	)

	body, err = ioutil.ReadAll(req.Body)

	if err != nil {
		str := fmt.Sprintf("[Handle] Unable to read in the body of the request: %s", body)
		log.Error(str)
		http.Error(writer, str, 500)
		return
	}

	err = json.Unmarshal(body, &work)

	if err != nil {
		str := fmt.Sprintf("[ERROR] Unable to parse workload JSON: %s", err)
		http.Error(writer, str, 500)
		return
	}

	err = process(work, &res)

	if err != nil {
		str := fmt.Sprintf("[ERROR] Problems processing workload: %s", err)
		http.Error(writer, str, 500)
		return
	}

	body, _ = json.Marshal(res)
	writer.Write(body)
}

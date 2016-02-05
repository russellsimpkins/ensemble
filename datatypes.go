package ensemble

import (
	"log"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	klog "github.com/go-kit/kit/log"
)

/*
 * Wouldn't it be cool to have a service you could call that would
 * make all of your api requests for you? This is for you. You simply
 * build up a json string that is all of your requests and post it to
 * this service. The service will run all of your requests for you and
 * give you a json response back. The packet of requests can be executed
 * in sequence if that is required, otherwise we will run the requests
 * async.
 */

// what if we could allow call dependencies? b depends on the result data of a, that sort of thing.
type Dependency struct {
	Request Request `json:"request"`
}

// a utility structure to pass one or more results to a request
type PassThruData struct {
	Data []interface{}
}

type Workload struct {
	Requests    []Request `json:"requests"`
	StrictOrder bool      `json:"strictorder"` // sync or async
	Timeout     int64     `json:"timeout"`     // TODO add a real timeout for the sync process
	UseHeaders  bool      `json:"use_headers"` // set this to true if the requests should use the headers of the work request
	header      http.Header
}

// our definition of a request
// i'm not sure if i should have ContentType or assume it's in the headers
type Request struct {
	Id           string       `json:"id"`           // some way to identify this request in the response
	URL          string       `json:"url"`          // the restful api to call
	Method       string       `json:"method"`       // request method: get/put/post/delete
	Data         string       `json:"data"`         // data to pass to api. if it's a get, we add ? to URL
	Header       http.Header  `json:"headers"`      // request specific headers to add
	Dependents   []Dependency `json:"dependency"`   // id of the request this request depends on
	UseData      bool         `json:"useData"`      // if so, Payload is sprintf-able
	UseDepHeader bool         `json:"useDepHeader"` // if you want to use the headers from dependent calls
	DepHeader    []string     `json:"DepHeaders"`   // name the headers to use.
	DoJoin       bool         `json:"doJoin"`       // you plan on passing multiple dependencies and need to join the results
	JoinChar     string       `json:"joinChar"`     // e.g. , or |
	PassByName   string       `json:"passName"`     // if it's not json, it parameterized and needs a name
}

type Response struct {
	Id     string      `json:"id"`   // some way to identify this request in the response
	Data   string      `json:"data"` // put the data here
	Object interface{} `json:"object"`
	Code   int         `json:"code"` // http response code
	Header http.Header `json:"headers"`
}

type Result struct {
	Responses []Response `json:"responses"`
	Err       string     `json:"err,omitempty"`
	Code      int        `json:"code"`
}

type Call struct {
	Req Request
	Res Response
}

type MagicService interface {
	// all magic comes in as a json blob and get's returned as a json blob
	DoMagic(Workload) (Result, error)
}

type Magic struct {
	logger *klog.Logger
}

// go-kit specifics

type Middleware func(endpoint.Endpoint) endpoint.Endpoint

type loggingMiddleware struct {
	logger log.Logger
	MagicService
}

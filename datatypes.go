package ensemble

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

// our definition of a request
// i'm not sure if i should have ContentType or assume it's in the headers
type Request struct {
	Id           string              `json:"id"`              // some way to identify this request in the response
	URL          string              `json:"url"`             // the restful api to call
	Method       string              `json:"method"`          // request method: get/put/post/delete
	Payload      string              `json:"payload"`         // data to pass to api. if it's a get, we add ? to URL
	Headers      map[string][]string `json:"headers"`         // request specific headers to add
	Depends      []Dependency        `json:"dependency"`      // id of the request this request depends on
	UseData      bool                `json:"useData"`         // if so, Payload is sprintf-able
	UseDepHeader bool                `json:"useDepHeader"`    // if you want to use the headers from dependent calls
	DepHeader    []string            `json:"DepHeaders"`      // name the headers to use.
	DoJoin       bool                `json:"doJoin"`          // you plan on passing multiple dependencies and need to join the results
	JoinChar     string              `json:"joinChar"`        // e.g. , or |
	EvalRespJson bool                `json:"evalJson"`        // Should we turn the text to json to avoid dbl encoding
	PassByName   string              `json:"passName"`        // if it's not json, it parameterized and needs a name
	ReqGraphics8 bool                `json:"graphics8Access"` // will create a auth cookie if true
	GraphicsPath string              `json:"graphicsPath"`    // what path are they trying to access e.g. "/mobile/v2/json/iphone/"
}

type ApiWrapper struct {
	Requests  []Request `json:"requests"`
	SyncOrder bool      `json:"strictorder"`
	Timeout   int64     `json:"timeout"`
}

type Response struct {
	Id      string      `json:"id"`      // some way to identify this request in the response
	Payload string      `json:"payload"` // put the data here
	Object  interface{} `json:"object"`
	Code    int         `json:"code"` // http response code
}

type Result struct {
	Responses []Response `json:"responses"`
}

Ensemble
==========
Ensemble is a service for combining aggregate RestFUL requests in one call. It's intended primarily for mobile or javascript developers who need to get send and receive data using multiple RestFUL requests.

To use this service you need to create you're own go http server e.g.

{code}
package main
import (
	"fmt"
	"github.com/russellsimpkins/ensemble"
	"net/http"
	"time"
)

func StartListener() {
	http.HandleFunc("/magic, ensemble.Handle)

	srv := &http.Server{
		Handler:        nil,
		Addr:           ":8080",
		WriteTimeout:   15 * time.Second,
		ReadTimeout:    15 * time.Second,
		MaxHeaderBytes: 32768,
	}
	
	err := srv.ListenAndServe()	
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}
}

func main() {
	StartListener()
}
{/code}

A lot of RestFUL APIs are written as CRUD services and this tool will let you glue API calls together.

Here is the most basic example.

{
    "requests": [{
        "id": "1",
        "url": "http://localhost/test1?foo=bar",
        "method": "GET"
    }, {
        "id": "2",
        "url": "http://localhost/test2",
        "method": "POST",
        "payload": "boo=far"
    }],
    "strictorder": false
}

The JSON specifies two calls and says that order is not important. The service will call each service and return the id, data and code. The id is there to help the caller identify each request. You can also have the service follow the order by setting "strictorder":true e.g.

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
    "strictorder": true
}

Assuming /test1 returns "This worked" and /test2 returns "That worked" your JSON response would look like this:

{
    "reponses": [{
        "id": "1",
        "payload": "This worked",
        "code": 200
    }, {
        "id": "2",
        "payload": "That worked",
        "code": 200
    }]
}

If /test2 returns JSON, it would look like this:  

{
    "reponses": [{
        "id": "1",
        "payload": "This worked",
        "code": 200
    }, {
        "id": "2",
        "payload": "{\"is this\":\"magic?\"},{\"yup\":\"magic happens\"}",,
        "code": 200
    }]
}

You can also add dependency calls into the mix. Let's say you want a couple of calls to be made before your call to test2, and you want that data sent along. Let's assume that /provide1 returns {"is this":"magic?"} and /provide2 returns {"yup":"magic happens"} which you wanted posted to /test2 and you created the following request:

{
    "requests": [{
        "id": "1",
        "url": "http://localhost:8080/test1",
        "method": "GET"
    }, {
        "id": "2",
        "url": "http://localhost:8080/test2",
        "method": "POST",
        "payload": "{\"data\":%s}",
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
}


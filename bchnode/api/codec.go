package api

import (
	"errors"
	"fmt"
	"net/http"

	"encoding/json"
	"github.com/gorilla/rpc"
)

type MyCodec struct {
}

// NewMyCodec returns a new MyCodec.
func NewMyCodec() *MyCodec {
	return &MyCodec{}
}

// NewRequest returns a new CodecRequest of type MyCodecRequest.
func (c *MyCodec) NewRequest(r *http.Request) rpc.CodecRequest {
	cr := new(MyCodecRequest) // Our custom CR
	req := new(serverRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	fmt.Printf("newRequest: %v, cr: %v\n", req, cr)
	_ = r.Body.Close()
	if err == nil {
		cr.serverRequest = req
	}
	return cr
}

type MyCodecRequest struct {
	*serverRequest
}

func (c *MyCodecRequest) Method() (string, error) {
	fmt.Printf("Method:%v\n", c)
	return c.serverRequest.Method + ".Call", nil
}

type JsonRpcError struct {
	Code    int `json:"code"`
	Message int `json:"message"`
}

type serverRequest struct {
	// A String containing the name of the method to be invoked.
	Method string `json:"method"`
	// An Array of objects to pass as arguments to the method.
	Params *json.RawMessage `json:"params"`
	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	Id *json.RawMessage `json:"id"`
}

type serverResponse struct {
	Result interface{}      `json:"result"`
	Error  interface{}      `json:"error"`
	Id     *json.RawMessage `json:"id"`
}

var null = json.RawMessage([]byte("null"))

func (c *MyCodecRequest) ReadRequest(args interface{}) error {
	var err error
	if c.Params != nil {
		params := [1]interface{}{args}
		err = json.Unmarshal(*c.Params, &params)
	} else {
		err = errors.New("rpc: method request ill-formed: missing params field")
	}
	return err
}

func (c *MyCodecRequest) WriteResponse(w http.ResponseWriter, reply interface{}, methodErr error) error {
	res := &serverResponse{
		Result: reply,
		Error:  &JsonRpcError{},
		Id:     c.serverRequest.Id,
	}
	if methodErr != nil {
		res.Error = JsonRpcError{
			Code:    -1,
			Message: -1,
		}
		res.Result = &null
	}
	var err error
	if c.serverRequest.Id == nil {
		res.Id = &null
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		encoder := json.NewEncoder(w)
		err = encoder.Encode(res)
	}
	return err
}

package main

import (
	"github.com/smartbch/testkit/bchnode/generator"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"

	"github.com/smartbch/testkit/bchnode/api"
)

func main() {
	generator.Init()
	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "text/plain")
	s.RegisterCodec(json.NewCodec(), "application/json")
	err := s.RegisterService(new(api.BlockCountService), "getblockcount")
	if err != nil {
		panic(err)
	}
	_ = s.RegisterService(new(api.BlockHashService), "getblockhash")
	_ = s.RegisterService(new(api.BlockService), "getblock")
	_ = s.RegisterService(new(api.TxService), "getrawtransaction")
	_ = s.RegisterService(new(api.PubKeyService), "pubkey")
	_ = s.RegisterService(new(api.BlockInternalService), "internal")

	r := mux.NewRouter()
	r.Handle("/", s)
	_ = http.ListenAndServe(":1234", r)
}

package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/smartbch/testkit/bchnode/api"
	"github.com/smartbch/testkit/bchnode/generator"
)

func main() {
	generator.Init()
	s := rpc.NewServer()
	s.RegisterCodec(api.NewMyCodec(), "text/plain")
	s.RegisterCodec(api.NewMyCodec(), "application/json")
	_ = s.RegisterService(new(api.BlockCountService), "getblockcount")
	_ = s.RegisterService(new(api.BlockHashService), "getblockhash")
	_ = s.RegisterService(new(api.BlockService), "getblock")
	_ = s.RegisterService(new(api.TxService), "getrawtransaction")
	_ = s.RegisterService(new(api.PubKeyService), "pubkey")
	_ = s.RegisterService(new(api.BlockIntervalService), "interval")
	_ = s.RegisterService(new(api.BlockReorgService), "reorg")
	_ = s.RegisterService(new(api.CCService), "cc")

	r := mux.NewRouter()
	r.Handle("/", s)
	_ = http.ListenAndServe(":1234", r)
}

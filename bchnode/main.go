package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/smartbch/testkit/bchnode/api"
	"github.com/smartbch/testkit/bchnode/generator"
)

func main() {
	ctx := generator.Init()
	s := rpc.NewServer()
	s.RegisterCodec(api.NewMyCodec(), "text/plain")
	s.RegisterCodec(api.NewMyCodec(), "application/json")
	_ = s.RegisterService(api.MakeBlockCountService(ctx), "getblockcount")
	_ = s.RegisterService(api.MakeBlockHashService(ctx), "getblockhash")
	_ = s.RegisterService(api.MakeBlockService(ctx), "getblock")
	_ = s.RegisterService(api.MakeTxService(ctx), "getrawtransaction")
	_ = s.RegisterService(api.MakePubKeyService(ctx), "pubkey")
	_ = s.RegisterService(api.MakeBlockIntervalService(ctx), "interval")
	_ = s.RegisterService(api.MakeBlockReorgService(ctx), "reorg")
	_ = s.RegisterService(api.MakeCCService(ctx), "cc")

	r := mux.NewRouter()
	r.Handle("/", s)
	_ = http.ListenAndServe(":1234", r)
}

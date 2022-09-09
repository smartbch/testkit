module github.com/smartbch/testkit

go 1.18

require (
	github.com/ethereum/go-ethereum v1.10.7
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/rpc v1.2.0
	github.com/smartbch/moeingads v0.4.0
	github.com/smartbch/moeingdb v0.4.0
	github.com/smartbch/moeingevm v0.4.0
	github.com/smartbch/smartbch v0.4.0
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.10
)

//replace github.com/smartbch/smartbch => ../smartbch

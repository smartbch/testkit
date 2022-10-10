## usage:

```
1. start single node network
cd ./../../smartbch
./restart_from_h0.sh
stop the node manually

2. build the fakenode
cd ./../bchnode
go build -o fakenode main.go

3. build the utxo tx maker
cd ./../bchutxomaker
go build -o txmaker main.go

4. build cc-operator
cd ./../../cc-operator
go build -o ccoperator main.go

5. prepare cc-contracts
cd ./../../cc-contracts
npm i

6. run test
cd ./../
npm i
truffle compile
go run main.go
```


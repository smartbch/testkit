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

4. run test
cd ./../
go run main.go
```


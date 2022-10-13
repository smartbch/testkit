pushd $PWD
echo 'prepare smartbch single node home dir'
cd ../../smartbch
git checkout testcc2
./restart_from_h0.sh
popd

pushd $PWD
echo 'build cc-operator'
cd ../../cc-operator
go build -o ccoperator main.go
popd

# pushd $PWD
# echo 'prepare cc-contracts'
# cd ../../cc-contracts
# npm i
# popd

pushd $PWD
echo 'build fakenode'
cd ../bchnode
go build -o fakenode main.go
popd

pushd $PWD
echo 'build txmaker'
cd ../bchutxomaker
go build -o txmaker main.go
popd

pushd $PWD
echo 'build fakecollector'
cd ../fakecollector
go build -o fakecollector github.com/smartbch/testkit/fakecollector
popd

# echo 'prepare truffle scripts'
# npm i
# truffle compile

file ../../smartbch/smartbchd
file ../../cc-operator/ccoperator
file ../bchnode/fakenode 
file ../bchutxomaker/txmaker
file ../fakecollector/fakecollector

echo 'run tests'
go run main.go

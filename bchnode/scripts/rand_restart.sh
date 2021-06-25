for ((i=0; i<1000; i++)); do
    ./smartbchd start --mainnet-url=http://34.88.14.23:1234 --smartbch-url=http://34.88.14.23:8545 --unlock=3462eacf9deccc36a8ef0dd51bc9d04a5fc2e354eac1ecb54eb2bfcb53788aa1 --watcher-speedup=true  --log-validators &
    let n=$RANDOM%200
    let n=$n+50
    echo "sleep $n start"
    sleep $n
    echo "sleep $n end"
    jobs
    kill -9 %1
    let n=$RANDOM%200
    let n=$n+50
    echo "SLEEP $n start"
    sleep $n
    echo "SLEEP $n end"
done


package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 6 {
		fmt.Println(`Usage: brokerchanger <rpcURL> <username> <password> <interval in seconds> <pubkey1=vp1> <pubkey2=vp2> ...

example:
brokerchanger http://47.115.171.70:28332/set_smartbch_brokerid.php regtest PLEXv-8D-FxWPMiXbspL 5 \
  0c54b6c7f07ebc6dfb6b192163c002dab348b56ce809688a222b3cf98eca36ee=1 \
  0c54b6c7f07ebc6dfb6b192163c002dab348b56ce809688a222b3cf98eca36ef=3`)
		return
	}

	bc := &BrokerChanger{
		rpcURL:   os.Args[1],
		username: os.Args[2],
		password: os.Args[3],
		interval: parseInterval(os.Args[4]),
		pubKeys:  getPubKeys(os.Args[5:]),
	}

	fmt.Println("rpcURL  : ", bc.rpcURL)
	fmt.Println("username: ", bc.username)
	fmt.Println("password: ", strings.Repeat("*", len(bc.password)))
	fmt.Println("interval: ", bc.interval, "(s)")
	fmt.Println("pubKeys : ", len(bc.pubKeys))

	bc.run()
}

func parseInterval(arg string) uint64 {
	i, err := strconv.ParseUint(arg, 10, 32)
	if err != nil {
		panic("invalid interval")
	}
	return i
}

func getPubKeys(args []string) (pks []PubKeyAndVP) {
	for _, arg := range args {
		ss := strings.Split(arg, "=")
		if len(ss) != 2 {
			panic("invalid arg: " + arg)
		}

		if len(ss[0]) != 64 {
			panic("invalid pubkey: " + ss[0])
		}
		if _, err := hex.DecodeString(ss[0]); err != nil {
			panic("invalid pubkey: " + ss[0])
		}

		vp, err := strconv.ParseUint(ss[1], 10, 32)
		if err != nil {
			panic("invalid voting power: " + ss[1])
		}

		pks = append(pks, PubKeyAndVP{
			pubKey:      ss[0],
			votingPower: vp,
		})
	}

	return
}

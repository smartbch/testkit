package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type PubKeyAndVP struct {
	pubKey      string
	votingPower uint64
}

type BrokerChanger struct {
	rpcURL   string
	username string
	password string
	interval uint64
	pubKeys  []PubKeyAndVP
	nextIdx  int
}

func (bc *BrokerChanger) run() {
	for {
		if bc.nextIdx >= len(bc.pubKeys) {
			bc.nextIdx = 0
		}

		pk := bc.pubKeys[bc.nextIdx]

		req := fmt.Sprintf(`{"params":["%s"]}`, pk.pubKey)
		fmt.Println(">>>", req)
		resp, err := bc.postJSON(req)
		fmt.Println("<<<", resp)
		if err == nil {
			bc.nextIdx++
		}

		time.Sleep(time.Duration(bc.interval*pk.votingPower) * time.Second)
	}
}

func (bc *BrokerChanger) postJSON(reqStr string) (string, error) {
	body := strings.NewReader(reqStr)
	req, err := http.NewRequest("POST", bc.rpcURL, body)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(bc.username, bc.password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	return string(respData), nil
}

package main

import (
	"encoding/json"
	"flag"
	"log"

	"github.com/icon-project/goloop/common/ipc"
	"github.com/pkg/errors"
)

type helloMessage struct {
	Success     bool
	BlockHeight uint
}

func Display(data interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func main() {
	var net, addr string

	flag.StringVar(&net, "net", "unix", "socket type")
	flag.StringVar(&addr, "addr", "/tmp/icon-rc.sock", "Address to connect")

	conn, err := ipc.Dial(net, addr)
	if err != nil {
		errors.Errorf("Failed to dial %s:%s err=%+v", net, addr, err)
	}

	var buf helloMessage
	forever := make(chan bool)
	conn.SendAndReceive(0, []byte("hello"), &buf)
	log.Printf("Result: %s", Display(buf))

	conn.Close()
	<-forever
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/icon-project/rewardcalculator/common/ipc"
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

func cmdUsage() {
	fmt.Printf("Usage: %s [[OPTIONS]] [COMMAND]\n", os.Args[0])
	fmt.Printf("OPTIONS\n")
	flag.PrintDefaults()
	fmt.Printf("COMMAND\n")
	fmt.Printf("\t hello         Send HELLO message\n")
	fmt.Printf("\t query         Send QUERY message with address to query I-Score\n")
	fmt.Printf("\t claim         Send CLAIM message with address to claim I-Score\n")
	fmt.Printf("\t calculate     Send CALCULATE message with IISS data path\n")
}

func main() {
	var command string
	var address string

	log.Printf("Get command: %s", os.Args)

	flag.StringVar(&address, "addr", "/tmp/icon-rc.sock", "I-Score database directory")
	flag.Usage = cmdUsage
	//queryCmd := flag.NewFlagSet("query", flag.ExitOnError)
	//queryAddress := queryCmd.String("address", "", "Account address(Required)")
	//claimCmd := flag.NewFlagSet("claim", flag.ExitOnError)
	//claimAddress := claimCmd.String("address", "", "Account address(Required)")
	//calculateCmd := flag.NewFlagSet("calculate", flag.ExitOnError)
	//calculateIISSData := calculateCmd.String("iiss-data", "", "IISS data DB path(Required)")
	flag.Parse()

	argc :=len(os.Args)
	if argc < 2 {
		flag.Usage()
		os.Exit(1)
	}
	command = os.Args[1]

	net := "unix"
	conn, err := ipc.Dial(net, address)
	if err != nil {
		fmt.Printf("Failed to dial %s:%s err=%+v\n", net, address, err)
		//os.Exit(1)
	}
	defer conn.Close()


	forever := make(chan bool)

	switch command {
	case "hello":
		if argc > 2 {
			fmt.Printf("'%s' command has no argument\n", command)
			os.Exit(1)
		}
		var buf helloMessage
		conn.SendAndReceive(0, []byte("hello"), &buf)
		fmt.Printf("Hello message result : %s", Display(buf))
	case "query":
	case "claim":
	case "calculate":
	default:
		fmt.Printf("Invalid COMMAND : %s", command)
		flag.Usage()
		os.Exit(1)
	}

	<-forever
}

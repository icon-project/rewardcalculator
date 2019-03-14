package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const (
	DBDir     = "/Users/eunsoopark/test/rc_test"
	DBType    = "goleveldb"
	DBName    = "test"
)

type CLI struct{}

func (cli *CLI) printUsage() {
	fmt.Printf("Usage: %s [db_name] [command]\n", os.Args[0])
	fmt.Printf("\t db_name     DB name\n")
	fmt.Printf("\t command     Command\n")
	fmt.Printf("\n [command]\n")
	fmt.Printf("\t create N NUM                 Create an I-Score DB with N Accout DBs and NUM accounts\n")
	fmt.Printf("\t delete                       Delete an I-Score DB\n")
	fmt.Printf("\t query KEY                    Query accounts value with KEY\n")
	fmt.Printf("\t calculate TO BATCH           Calculate I-Score of all account\n")
	fmt.Printf("\t           TO                 Block height to calculate. Set 0 if you want current block+1\n")
	fmt.Printf("\t           BATCH              The number of DB write batch count\n")
	fmt.Printf("\t issdata PATH                 Read IISS data DB\n")
	fmt.Printf("\t           PATH               Directory where the IISS data DB is located\n")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 3 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	cli.validateArgs()

	dbName := os.Args[1]
	cmd := os.Args[2]

	// Initialize the CLI
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	queryCmd := flag.NewFlagSet("query", flag.ExitOnError)
	calculateCmd := flag.NewFlagSet("calculate", flag.ExitOnError)
	iissDataCmd := flag.NewFlagSet("iissdata", flag.ExitOnError)

	createDBCount := createCmd.Int("db", 16, "The number of RC Account DB")
	createAccountCount := createCmd.Int("account", 10000000, "The account number of RC Account DB")
	queryAddress := queryCmd.String("address", "", "Account address")
	calculateBlockHeight := calculateCmd.Uint64("block", 0, "Block height to calculate, Set 0 if you want current block +1")
	calculateWriteBatch := calculateCmd.Uint64("writebatch", 0, "The number of DB write batch count")
	iissDataPath := iissDataCmd.String("path", "./", "Directory where the IISS data DB is located")

	// Parse the CLI
	switch cmd {
	case "create":
		err := createCmd.Parse(os.Args[3:])
		if err != nil {
			createCmd.Usage()
			os.Exit(1)
		}
	case "delete":
		err := deleteCmd.Parse(os.Args[3:])
		if err != nil {
			deleteCmd.PrintDefaults()
			os.Exit(1)
		}
	case "query":
		err := queryCmd.Parse(os.Args[3:])
		if err != nil {
			queryCmd.Usage()
			os.Exit(1)
		}
	case "calculate":
		err := calculateCmd.Parse(os.Args[3:])
		if err != nil {
			calculateCmd.Usage()
			os.Exit(1)
		}
	case "iissdata":
		err := iissDataCmd.Parse(os.Args[3:])
		if err != nil {
			iissDataCmd.Usage()
			os.Exit(1)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	// Run the command
	if createCmd.Parsed() {
		if *createDBCount <= 0 || *createAccountCount <= 0 {
			createCmd.Usage()
			os.Exit(1)
		}

		start := time.Now()

		// create
		cli.create(dbName, *createDBCount, *createAccountCount)

		end := time.Now()
		diff := end.Sub(start)
		fmt.Printf("Duration : %v\n", diff)
	}

	if deleteCmd.Parsed() {
		cli.delete(dbName)
	}

	if queryCmd.Parsed() {
		if *queryAddress == "" {
			queryCmd.Usage()
			os.Exit(1)
		}
		cli.query(dbName, *queryAddress)
	}

	if calculateCmd.Parsed() {
		start := time.Now()

		// calculate
		cli.calculate(dbName, *calculateBlockHeight, *calculateWriteBatch)

		end := time.Now()
		diff := end.Sub(start)
		fmt.Printf("Duration : %v\n", diff)
	}

	if iissDataCmd.Parsed() {
		cli.iissData(*iissDataPath, dbName)
	}
}

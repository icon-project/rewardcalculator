package main

import "os"

func main() {
	cli := CLI{}
	ret := cli.Run()
	if ret {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

package main

import "fmt"

func main() {
	if err := Run(); err != nil {
		fmt.Println(err.Error())
	}
}

package main

import "os"

func main() {
	port := "3333"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	runServer(port)
}

package main

import "github.com/hysios/mx"

func main() {
	(&mx.Gateway{}).Serve(":8080")
}

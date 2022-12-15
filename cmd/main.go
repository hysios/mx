package main

import (
	_ "embed"
	"strings"

	"os"

	"github.com/hnhuaxi/appcli"
)

//go:embed app.yaml
var appSource string

var app = appcli.App{
	Source: strings.NewReader(appSource),
}

func main() {
	_ = app.Execute(os.Args)
}

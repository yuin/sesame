// package main is an executable command for sasame(https://github.com/yuin/sesame),
// an object to object mapper for Go.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/yuin/sesame"
)

func main() {
	if len(os.Getenv("DEBUG")) != 0 {
		sesame.LogEnabledFor = sesame.LogLevelDebug
	}

	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	generateConfig := generateCmd.String("c", "sesame.yml", "config file path")
	generateHelp := generateCmd.Bool("h", false, "show this help")
	generateQuiet := generateCmd.Bool("q", false, "suppress messages")

	cmdName := "generate"
	args := []string{}
	if len(os.Args) > 1 {
		cmdName = os.Args[1]
		args = os.Args[2:]
	}
redo:

	switch cmdName {
	case "generate":
		err := generateCmd.Parse(args)
		if err != nil {
			sesame.LogFunc(sesame.LogLevelError, err.Error())
			os.Exit(1)
		}
		if *generateHelp {
			generateCmd.Usage()
			os.Exit(1)
		}
		if *generateQuiet {
			sesame.LogEnabledFor = sesame.LogLevelError
		}
		var config sesame.Generation
		if err := sesame.LoadConfig(&config, *generateConfig); err != nil {
			sesame.LogFunc(sesame.LogLevelError, err.Error())
			os.Exit(1)
		}
		if sesame.LogEnabledFor >= sesame.LogLevelDebug {
			b, _ := json.Marshal(&config)
			sesame.LogFunc(sesame.LogLevelDebug, string(b))
		}
		generator := sesame.NewGenerator(&config)
		if err := generator.Generate(); err != nil {
			sesame.LogFunc(sesame.LogLevelError, err.Error())
			os.Exit(1)
		}
	case "-h":
		fmt.Fprint(os.Stderr, `sesame [COMMAND|-h]
  COMMANDS:
    generate: generates mappers(default)
  OPTIONS:
    -h: show this help
`)
		os.Exit(1)
	default:
		cmdName = "generate"
		args = os.Args[1:]
		goto redo
	}
}

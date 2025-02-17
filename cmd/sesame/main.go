// package main is an executable command for sasame(https://github.com/yuin/sesame),
// an object to object mapper for Go.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	sesameinternal "github.com/yuin/sesame/internal"
)

func main() {
	if len(os.Getenv("DEBUG")) != 0 {
		sesameinternal.LogEnabledFor = sesameinternal.LogLevelDebug
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
			sesameinternal.LogFunc(sesameinternal.LogLevelError, err.Error())
			os.Exit(1)
		}
		if *generateHelp {
			generateCmd.Usage()
			os.Exit(1)
		}
		if *generateQuiet {
			sesameinternal.LogEnabledFor = sesameinternal.LogLevelError
		}
		var config sesameinternal.Generation
		if err := sesameinternal.LoadConfig(&config, *generateConfig); err != nil {
			sesameinternal.LogFunc(sesameinternal.LogLevelError, err.Error())
			os.Exit(1)
		}
		if sesameinternal.LogEnabledFor >= sesameinternal.LogLevelDebug {
			b, _ := json.Marshal(&config)
			sesameinternal.LogFunc(sesameinternal.LogLevelDebug, string(b))
		}
		generator := sesameinternal.NewGenerator(&config)
		if err := generator.Generate(); err != nil {
			sesameinternal.LogFunc(sesameinternal.LogLevelError, err.Error())
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

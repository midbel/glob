package main

import (
	"flag"
	"fmt"
  "os"
  "strings"

	"github.com/midbel/glob"
)

func main() {
	var (
		matching  = flag.Bool("m", false, "matching")
		compiling = flag.Bool("c", false, "compiling")
	)
	flag.Parse()

	var err error
	switch {
	case *compiling:
		err = runCompile(flag.Args())
	case *matching:
		err = runMatch(flag.Args())
	default:
		args := make([]string, flag.NArg()-1)
		for i := 1; i < flag.NArg(); i++ {
			args[i-1] = flag.Arg(i)
		}
		err = runGlob(flag.Arg(0), args)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runMatch(args []string) error {
	for i := 1; i < len(args); i++ {
		err := glob.Match(args[i], args[0])
		if err != nil {
			fmt.Println("no match:", args[i])
		}
	}
	return nil
}

func runCompile(args []string) error {
	for i := 0; i < len(args); i++ {
		if i > 0 {
			fmt.Println("--")
		}
		m, err := glob.Compile(strings.TrimSpace(args[i]))
		if err != nil {
			return err
		}
		glob.Debug(m)
	}
	return nil
}

func runGlob(pattern string, base []string) error {
	if len(base) == 0 {
		return fmt.Errorf("no directory given")
	}
	g, err := glob.New(pattern, base...)
	if err != nil {
		return err
	}
	for f := g.Glob(); f != ""; f = g.Glob() {
		fmt.Println(f)
	}
	return nil
}

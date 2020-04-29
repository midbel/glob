package main

import (
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"github.com/midbel/glob"
	"github.com/midbel/linewriter"
	"github.com/midbel/sizefmt"
	"github.com/midbel/xxh"
)

func main() {
	var (
		matching  = flag.Bool("m", false, "matching")
		compiling = flag.Bool("c", false, "compiling")
		csv       = flag.Bool("p", false, "csv")
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
		err = runGlob(flag.Arg(0), args, *csv)
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

func runGlob(pattern string, base []string, csv bool) error {
	var option linewriter.Option
	if csv {
		option = linewriter.AsCSV(false)
	} else {
		option = linewriter.WithPadding([]byte(" "))
	}
	g, err := glob.New(pattern, base...)
	if err != nil {
		return err
	}
	var (
		total  float64
		files  uint
		line   = linewriter.NewWriter(4096, option)
		digest = xxh.New64(0)
	)
	for f := g.Glob(); f != ""; f = g.Glob() {
		size, sum, err := statFile(f, digest)
		if err != nil {
			return err
		}

		line.AppendSize(size, 10, linewriter.SizeIEC)
		line.AppendBytes(sum, 16, linewriter.Hex)
		line.AppendString(f, 0, linewriter.AlignLeft)
		if _, err := io.Copy(os.Stdout, line); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		total += float64(size)
		files++
	}
	fmt.Printf("%d files (%s)\n", files, sizefmt.Format(total, sizefmt.IEC))
	return nil
}

func statFile(f string, digest hash.Hash) (int64, []byte, error) {
	r, err := os.Open(f)
	if err != nil {
		return 0, nil, err
	}
	defer func() {
		r.Close()
		digest.Reset()
	}()

	if _, err := io.Copy(digest, r); err != nil {
		return 0, nil, err
	}
	s, err := r.Stat()
	if err != nil {
		return 0, nil, err
	}
	return s.Size(), digest.Sum(nil), nil
}

package main

import (
	"context"
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
	"golang.org/x/sync/semaphore"
)

func main() {
	var (
		matching  = flag.Bool("m", false, "matching")
		compiling = flag.Bool("c", false, "compiling")
		fast      = flag.Bool("f", false, "fast")
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
		err = runGlob(flag.Arg(0), args, *fast, *csv)
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

type FileInfo struct {
	File string
	Hash []byte
	Size int64
	Err  error
}

func runGlob(pattern string, base []string, fast, csv bool) error {
	var option linewriter.Option
	if csv {
		option = linewriter.AsCSV(false)
	} else {
		if !fast {
			option = linewriter.WithPadding([]byte(" "))
		} else {
			option = linewriter.WithPadding([]byte(""))
		}
	}
	g, err := glob.New(pattern, base...)
	if err != nil {
		return err
	}
	var (
		total float64
		files uint
		line  = linewriter.NewWriter(4096, option)
	)
	for fi := range gatherInfos(g, fast) {
		if fi.Err != nil {
			return fi.Err
		}
		if !fast {
			line.AppendSize(fi.Size, 10, linewriter.SizeIEC)
			line.AppendBytes(fi.Hash, 16, linewriter.Hex)
		}
		line.AppendString(fi.File, 0, linewriter.AlignLeft)
		if _, err := io.Copy(os.Stdout, line); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		total += float64(fi.Size)
		files++
	}
	fmt.Printf("%d files (%s)\n", files, sizefmt.Format(total, sizefmt.IEC))
	return nil
}

func gatherInfos(g *glob.Glob, fast bool) <-chan FileInfo {
	queue := make(chan FileInfo)
	go func() {
		defer close(queue)
		var (
			ctx    = context.TODO()
			sema   = semaphore.NewWeighted(16)
			digest = xxh.New64(0)
		)
		for f := g.Glob(); f != ""; f = g.Glob() {
			sema.Acquire(ctx, 1)
			go func(file string) {
				defer sema.Release(1)
				fi, err := statFile(file, digest, fast)
				fi.Err = err

				queue <- fi
			}(f)
		}
		sema.Acquire(ctx, 16)
	}()
	return queue
}

func statFile(f string, digest hash.Hash, fast bool) (FileInfo, error) {
	var fi FileInfo

	r, err := os.Open(f)
	if err != nil {
		return fi, err
	}
	defer func() {
		r.Close()
		digest.Reset()
	}()

	if !fast {
		if _, err := io.Copy(digest, r); err != nil {
			return fi, err
		}
	}
	s, err := r.Stat()
	if err != nil {
		return fi, err
	}

	fi.File = f
	fi.Size = s.Size()
	fi.Hash = append(fi.Hash, digest.Sum(nil)...)
	return fi, nil
}

package glob

import (
	"io"
	"os"
	"path/filepath"
)

func init() {
	scan = scandir
}

type Glob struct {
	queue   <-chan entry
	keepDir bool
}

func New(pattern string, dirs ...string) (*Glob, error) {
	m, err := Compile(pattern)
	if err != nil {
		return nil, err
	}
	queue := make(chan entry)
	go func() {
		defer close(queue)
		for _, d := range dirs {
			glob(d, m, queue)
		}
	}()
	return &Glob{queue: queue}, nil
}

func (g *Glob) Glob() string {
	for {
		e := <-g.queue
		if !g.keepDir && e.Dir {
			continue
		}
		return e.Name
	}
}

func glob(dir string, m Matcher, q chan<- entry) {
	if m == nil {
		return
	}
	es, err := scan(dir)
	if err != nil {
		return
	}
	for e := range es {
		next, err := m.Match(e.Name)
		if err != nil {
			continue
		}
		file := filepath.Join(dir, e.Name)
		if e.Dir {
			glob(file, next, q)
		}
		if next == nil {
			q <- entry{
				Name: file,
				Dir:  e.Dir,
			}
		}
	}
}

type entry struct {
	Name string
	Dir  bool
}

var scan func(string) (<-chan entry, error)

func scandir(dir string) (<-chan entry, error) {
	r, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	queue := make(chan entry)
	go func() {
		defer func() {
			close(queue)
			r.Close()
		}()
		for {
			is, err := r.Readdir(64)
			if len(is) == 0 || err == io.EOF {
				break
			}
			for _, i := range is {
				if set := i.Mode() & os.ModeSymlink; set != 0 {
					f, err := filepath.EvalSymlinks(filepath.Join(dir, i.Name()))
					if err != nil {
						continue
					}
					if i, err = os.Stat(f); err != nil {
						continue
					}
				}
				queue <- entry{
					Name: i.Name(),
					Dir:  i.IsDir(),
				}
			}
		}
	}()
	return queue, nil
}

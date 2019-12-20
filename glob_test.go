package glob

import (
	"testing"
)

func init() {
	scan = scanmap
}

func TestGlob(t *testing.T) {
	t.SkipNow()

}

func scanmap(_ string) (<-chan entry, error) {
	queue := make(chan entry)
	go func() {
		defer close(queue)
	}()
	return queue, nil
}

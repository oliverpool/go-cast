package discover_test

import (
	"testing"

	"github.com/barnybug/go-cast/discover"

	"github.com/barnybug/go-cast"
	"golang.org/x/net/context"

	"github.com/barnybug/go-cast/mock"
)

func TestFirstDirect(t *testing.T) {
	scan := mock.Scanner{
		ScanFunc: func(ctx context.Context, results chan<- *cast.Chromecast) error {
			results <- &cast.Chromecast{}
			close(results)
			return nil
		},
	}

	service := discover.Service{Scanner: &scan}

	ctx := context.Background()

	first, err := service.First(ctx)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if first == nil {
		t.Errorf("a client should have been found")
	}
	if scan.ScanFuncCalled != 1 {
		t.Errorf("scanner should have been called once, and not %d times", scan.ScanFuncCalled)
	}
}

func TestFirstCancelled(t *testing.T) {
	scan := mock.Scanner{
		ScanFunc: func(ctx context.Context, results chan<- *cast.Chromecast) error {
			<-ctx.Done()
			return nil
		},
	}

	service := discover.Service{Scanner: &scan}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	first, err := service.First(ctx)
	if err != ctx.Err() {
		t.Errorf("unexpected error %v", err)
	}
	if first != nil {
		t.Errorf("a client should not have been found")
	}
	if scan.ScanFuncCalled > 1 {
		t.Errorf("scanner should have been called at most once, and not %d times", scan.ScanFuncCalled)
	}
}

func TestNamedDirect(t *testing.T) {
	scan := mock.Scanner{}
	done := make(chan struct{})
	scan.ScanFunc = func(ctx context.Context, results chan<- *cast.Chromecast) error {
		defer close(results)
		results <- &cast.Chromecast{}
		c := &cast.Chromecast{
			Info: map[string]string{
				"fn": "casti",
			},
		}
		results <- c
		results <- &cast.Chromecast{}
		select {
		case results <- &cast.Chromecast{}:
			t.Error("channel should have been full")
		case <-ctx.Done():
		}
		close(done)
		return nil
	}

	service := discover.Service{Scanner: &scan}

	ctx := context.Background()

	first, err := service.Named(ctx, "casti")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if first == nil {
		t.Errorf("a client should have been found")
	}
	if first.Name() != "casti" {
		t.Errorf("the client should been named 'casti' and not '%s'", first.Name())
	}
	if scan.ScanFuncCalled != 1 {
		t.Errorf("scanner should have been called once, and not %d times", scan.ScanFuncCalled)
	}
	<-done
}

func TestNamedCancelled(t *testing.T) {
	scan := mock.Scanner{}
	done := make(chan struct{})
	scan.ScanFunc = func(ctx context.Context, results chan<- *cast.Chromecast) error {
		defer close(results)
		for {
			select {
			case results <- &cast.Chromecast{}:
			case <-ctx.Done():
				close(done)
				return nil
			}
		}
	}

	service := discover.Service{Scanner: &scan}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	first, err := service.Named(ctx, "casti")
	if err != ctx.Err() {
		t.Errorf("unexpected error %v", err)
	}
	if err != ctx.Err() {
		t.Errorf("unexpected error %v", err)
	}
	if first != nil {
		t.Errorf("a client should not have been found")
	}
	if scan.ScanFuncCalled > 1 {
		t.Errorf("scanner should have been called at most once, and not %d times", scan.ScanFuncCalled)
	}
	<-done
}

func TestUniq(t *testing.T) {
	in := make(chan *cast.Chromecast, 10)
	in <- &cast.Chromecast{}
	in <- &cast.Chromecast{}
	in <- &cast.Chromecast{}
	in <- &cast.Chromecast{}
	c := &cast.Chromecast{
		Info: map[string]string{
			"id": "123",
		},
	}
	in <- c
	in <- c
	close(in)

	out := make(chan *cast.Chromecast, 2)
	discover.Uniq(in, out)
	c = <-out
	if c.ID() != "" {
		t.Errorf("unexpected ID: %s", c.ID())
	}
	c = <-out
	if c.ID() != "123" {
		t.Errorf("unexpected ID: %s", c.ID())
	}
	c, ok := <-out
	if ok {
		t.Error("out should have been closed")
	}
}

package marx // import "github.com/keks/marx"

import (
	"context"
	"errors"
	"sync"

	"github.com/hashicorp/go-multierror"
)

type Worker func(context.Context) error

func Unite(wfs ...Worker) Worker {
	return Worker(func(ctx context.Context) error {
		var (
			wg   sync.WaitGroup
			l    sync.Mutex
			errs error
		)

		wg.Add(len(wfs))

		for _, wf := range wfs {
			go func(wf Worker) {
				defer wg.Done()

				err := wf(ctx)
				if err != nil {
					l.Lock()
					errs = multierror.Append(errs, err)
					l.Unlock()
				}
			}(wf)
		}

		wg.Wait()

		return errs
	})
}

func NewUnion() (Worker, func(Worker) error) {
	var (
		ch     = make(chan Worker)
		waitCh = make(chan struct{})

		wg   sync.WaitGroup
		l    sync.Mutex
		once sync.Once
		errs error
	)

	add := func(w Worker) error {
		l.Lock()
		defer l.Unlock()

		select {
		case <-waitCh:
			return errors.New("union disbanded")
		default:
		}

		wg.Add(1)
		ch <- w

		return nil
	}

	blockWorker := func(ctx context.Context) error {
		run := func(ctx context.Context, w Worker) {
			var err error

			defer func() {
				l.Lock()
				if err != nil {
					errs = multierror.Append(errs, err)
				}
				wg.Done()
				l.Unlock()
			}()

			err = w(ctx)
		}

		waiter := func() {
			wg.Wait()
			l.Lock()
			defer l.Unlock()
			close(waitCh)
		}

		for {
			select {
			case <-waitCh:
				return errs
			case w := <-ch:
				once.Do(func() {
					go waiter()
				})

				go run(ctx, w)

			}
		}
	}

	return blockWorker, add
}

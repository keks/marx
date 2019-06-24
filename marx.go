package marx // import "github.com/keks/marx"

import (
	"context"
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

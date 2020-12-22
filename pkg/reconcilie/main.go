package reconcilie

import (
	"context"
	"fmt"
)

var tasks = []struct {
	Name        string
	Do          func(context.Context, Params) error
	BailOnError bool
}{
	{
		Name:        "collector",
		Do:          Collector,
		BailOnError: true,
	},
}

func Run(ctx context.Context, params Params) error {
	for _, task := range tasks {
		if err := task.Do(ctx, params); err != nil {
			params.Log.Error(err, fmt.Sprintf("failed to reconcile %s", task.Name))
			if task.BailOnError {
				return err
			}
		}
	}
	return nil
}

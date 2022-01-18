package parmap

import (
	"context"
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/MaxPower15/govips-segfault/slice"
)

type indexedInterface struct {
	Index int
	Value interface{}
	Error error
}

type indexedInterfaceByIndex []indexedInterface

func (a indexedInterfaceByIndex) Len() int           { return len(a) }
func (a indexedInterfaceByIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a indexedInterfaceByIndex) Less(i, j int) bool { return a[i].Index < a[j].Index }

func IntsToStrings(ctx context.Context, items []int, workers int, fn func(int, int) (string, error)) ([]string, error) {
	inputs, err := Inputs(items)
	if err != nil {
		return []string{}, fmt.Errorf("making inputs: %w", err)
	}
	results, err := InterfacesToInterfaces(ctx, inputs, workers, func(index int, item interface{}) (interface{}, error) {
		if typedItem, ok := item.(int); ok {
			return fn(index, typedItem)
		} else {
			return "", fmt.Errorf("item cannot be cast to int: %v", item)
		}
	})
	if err != nil {
		return []string{}, fmt.Errorf("parmap.parmapIntsToStrings: %w", err)
	}

	typedResults := []string{}
	for _, item := range results {
		if typedItem, ok := item.(string); ok {
			typedResults = append(typedResults, typedItem)
		} else {
			return typedResults, fmt.Errorf("item cannot be cast to string: %v", item)
		}
	}

	return typedResults, err
}

func InterfacesToInterfaces(ctx context.Context, inputs []interface{}, workers int, fn func(int, interface{}) (interface{}, error)) ([]interface{}, error) {
	if workers < 1 {
		return []interface{}{}, fmt.Errorf("must have at least one worker, specified %d", workers)
	}

	if len(inputs) == 0 {
		return []interface{}{}, nil
	}

	jobsChan := make(chan indexedInterface, len(inputs))
	defer func() {
		// explicitly closing jobsChan ensures that the worker goroutines all
		// exit when we're done. otherwise they'll wait forever and eventually
		// we'll hit our goroutine limit.
		close(jobsChan)
	}()

	outputChan := make(chan indexedInterface, len(inputs))

	errorCount := uint64(0)

	// fire up our workers. each one will pull a job off jobsChan if it's not
	// busy. first one to pull it off gets to process it. when it finishes, it
	// pushes the output onto another channel.
	for i := 0; i < workers; i++ {
		go func() {
			for job := range jobsChan {
				select {
				case <-ctx.Done():
					return
				default:
					if atomic.LoadUint64(&errorCount) > 0 {
						// we may already have pulled in more jobs to execute before
						// we've identified the error and closed the channel. to
						// prevent us from executing jobs immediately, we'll atomicly
						// check if there are any errors.
						return
					}
					output, err := fn(job.Index, job.Value)
					if err != nil {
						atomic.AddUint64(&errorCount, 1)
					}
					outputChan <- indexedInterface{Index: job.Index, Value: output, Error: err}
				}
			}
		}()
	}

	// our workers are all raring to go; put all of our jobs into jobsChan so they
	// can take 'em as fast as possible. this should not block because the size of
	// jobsChan buffer is the same as len(inputs).
	for index, input := range inputs {
		jobsChan <- indexedInterface{Index: index, Value: input}
	}

	// wait for either the context to be done or until we receive all the outputs
	// we expect. if we get an error, we can bail early.
	indexedOutputs := []indexedInterface{}
	stillWaitingForOutput := true
	for stillWaitingForOutput {
		select {
		case <-ctx.Done():
			return []interface{}{}, ctx.Err()
		case output := <-outputChan:
			if output.Error != nil {
				return []interface{}{}, fmt.Errorf("index %d: %w", output.Index, output.Error)
			}
			indexedOutputs = append(indexedOutputs, output)
			if len(indexedOutputs) >= len(inputs) {
				stillWaitingForOutput = false
			}
		}
	}

	// the caller doesn't expect to receive the indexedInterface type; they expect to
	// receive the type their function returned. and they expect the output to be
	// in the same order as the output. so we sort, and then pull the actual
	// values out as return values for the caller.
	sort.Sort(indexedInterfaceByIndex(indexedOutputs))
	var outputs []interface{}
	for _, indexedOutput := range indexedOutputs {
		outputs = append(outputs, indexedOutput.Value)
	}

	return outputs, nil
}

func Inputs(s interface{}) ([]interface{}, error) {
	return slice.ToInterfaceSlice(s)
}

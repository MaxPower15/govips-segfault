package parmap

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wistia/render-pipeline/rpjson"
)

func parmapIntsToStrings(ctx context.Context, items []int, workers int, fn func(int, int) (string, error)) ([]string, error) {
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

func sameStringSlice(x, y []string) bool {
	return rpjson.Compact(x) == rpjson.Compact(y)
}

func TestParMap_basic_functionality(t *testing.T) {
	t.Run("workers = 1 succeeds", func(t *testing.T) {
		ctx := context.Background()
		numbers := []int{0, 1, 2, 3}
		results, err := parmapIntsToStrings(ctx, numbers, 1, func(index, item int) (string, error) {
			return strconv.Itoa(item), nil
		})
		if err != nil {
			t.Errorf("got error: %s", err)
		}

		if !sameStringSlice(results, []string{"0", "1", "2", "3"}) {
			t.Errorf("unexpected result %s", results)
		}
	})

	t.Run("workers gt 0 lt inputs succeeds", func(t *testing.T) {
		ctx := context.Background()
		numbers := []int{0, 1, 2, 3}
		results, err := parmapIntsToStrings(ctx, numbers, 3, func(index, item int) (string, error) {
			return strconv.Itoa(item), nil
		})
		if err != nil {
			t.Errorf("got error: %s", err)
		}

		if !sameStringSlice(results, []string{"0", "1", "2", "3"}) {
			t.Errorf("unexpected result %s", results)
		}
	})

	t.Run("workers gt inputs succeeds", func(t *testing.T) {
		ctx := context.Background()
		numbers := []int{0, 1, 2, 3}
		results, err := parmapIntsToStrings(ctx, numbers, 10, func(index, item int) (string, error) {
			return strconv.Itoa(item), nil
		})
		if err != nil {
			t.Errorf("got error: %s", err)
		}

		if !sameStringSlice(results, []string{"0", "1", "2", "3"}) {
			t.Errorf("unexpected result %s", results)
		}
	})

	t.Run("workers = 0 errors out", func(t *testing.T) {
		ctx := context.Background()
		numbers := []int{0, 1, 2, 3}
		_, err := parmapIntsToStrings(ctx, numbers, 0, func(index, item int) (string, error) {
			return strconv.Itoa(item), nil
		})
		if err == nil {
			t.Errorf("expected an error")
		}
		if err != nil && !strings.Contains(err.Error(), "worker") {
			t.Errorf("expected an error about the number of workers")
		}
	})
}

func TestParMap_is_parallel(t *testing.T) {
	t.Run("actually running in parallel", func(t *testing.T) {
		ctx := context.Background()
		numbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		started := time.Now().UnixMilli()
		_, err := parmapIntsToStrings(ctx, numbers, len(numbers), func(index, item int) (string, error) {
			time.Sleep(1 * time.Second)
			return strconv.Itoa(item), nil
		})
		ended := time.Now().UnixMilli()
		if err != nil {
			t.Errorf("got error: %s", err)
		}

		if ended-started > 2000 {
			// giving a lot of leniency to allow for system GC or whatever. we usually expect this
			// to be much closer to 1000ms.
			t.Errorf("not running as fast as expected")
		}
	})
}

func TestParMap_error_reports_no_races(t *testing.T) {
	t.Run("error returned from one function, reports error", func(t *testing.T) {
		// there is a default limit of 8128 simultaneous allowed goroutines. we intentionally
		// start 4100 here to make sure, when parmap finishes, it is cleanly closing its
		// goroutines. otherwise we'll go over the limit.
		//
		// running it a bunch of times also ensures that we will always catch an error if it's
		// thrown. this was introduced because there were some race conditions where we would
		// finish before receiving the error, and it would end up getting ignored.
		for i := 0; i < 4100; i++ {
			ctx := context.Background()
			numbers := []int{0, 1, 2, 3}
			_, err := parmapIntsToStrings(ctx, numbers, 2, func(index, item int) (string, error) {
				if item == 3 {
					return "", fmt.Errorf("not a fan of 3")
				}
				return strconv.Itoa(item), nil
			})
			if err == nil {
				t.Errorf("expected an error")
			}
			testStr := "not a fan of 3"
			if err != nil && !strings.Contains(err.Error(), testStr) {
				t.Errorf("expected '%s' to be included but got '%s'", testStr, err)
			}
		}
	})
}

func TestParMap_context_can_cancel(t *testing.T) {
	t.Run("context cancels, no more jobs executed", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancelFn := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancelFn()
		numbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		ran := uint64(0)
		_, err := parmapIntsToStrings(ctx, numbers, 1, func(index, item int) (string, error) {
			atomic.AddUint64(&ran, 1)
			time.Sleep(30 * time.Millisecond)
			return strconv.Itoa(item), nil
		})
		if err == nil {
			t.Errorf("expected context timeout error, got nil")
		}
		ran = atomic.LoadUint64(&ran)
		if ran == 0 {
			t.Errorf("expected to run at least once")
		}
		if ran > 2 {
			t.Errorf("expected to run no more than twice before being cancelled, ran %d times", ran)
		}
	})
}

func TestParMap_error_stops_more_jobs(t *testing.T) {
	t.Run("error cancels, no more jobs executed", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancelFn := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancelFn()
		numbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		ran := uint64(0)
		_, err := parmapIntsToStrings(ctx, numbers, 1, func(index, item int) (string, error) {
			atomic.AddUint64(&ran, 1)
			if item == 2 {
				return "hi", fmt.Errorf("2 is bad")
			}
			time.Sleep(100 * time.Millisecond)
			return strconv.Itoa(item), nil
		})
		if err == nil {
			t.Errorf("expected context timeout error, got nil")
		}
		ran = atomic.LoadUint64(&ran)
		if ran == 0 {
			t.Errorf("expected to run at least once")
		}
		if ran > 2 {
			t.Errorf("expected to run no more than twice before being cancelled, ran %d times", ran)
		}
	})
}

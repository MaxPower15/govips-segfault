package seq

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSeq_functionality(t *testing.T) {
	t.Run("Next() increases the value each time it's called", func(t *testing.T) {
		s := Seq(0)
		if s.Next() != 1 {
			t.Errorf("expected value to be 1")
		}
		if s.Next() != 2 {
			t.Errorf("expected value to be 2")
		}
		if s.Next() != 3 {
			t.Errorf("expected value to be 3")
		}
	})

	t.Run("Reset() sets the value back to 0", func(t *testing.T) {
		s := Seq(0)
		var val uint64
		s.Next()
		val = s.Next()
		if val != 2 {
			t.Errorf("expected value to be 2, got %d", val)
		}
		s.Reset(0)
		val = s.Next()
		if val != 1 {
			t.Errorf("expected value to be 1, got %d", val)
		}
	})

	t.Run("NextAsInt() returns an int type", func(t *testing.T) {
		s := Seq(0)
		var val int
		val = s.NextAsInt()
		if val != 1 {
			t.Errorf("expected value to be 1, got %d", val)
		}
		val = s.NextAsInt()
		if val != 2 {
			t.Errorf("expected value to be 2, got %d", val)
		}
	})

	t.Run("NextAsString() returns a string type", func(t *testing.T) {
		s := Seq(0)
		var val string
		val = s.NextAsString()
		if val != "1" {
			t.Errorf("expected value to be 1, got %s", val)
		}
		val = s.NextAsString()
		if val != "2" {
			t.Errorf("expected value to be 2, got %s", val)
		}
	})
}

func TestSeq_no_parallel_collisions(t *testing.T) {
	s := Seq(0)
	var wg sync.WaitGroup

	concurrency := 10000

	resultsChan := make(chan uint64, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(20 * time.Millisecond)
			resultsChan <- s.Next()
		}()
	}
	fmt.Println("waiting")
	wg.Wait()
	fmt.Println("done waiting")

	resultsMap := map[uint64]bool{}
	for i := 0; i < concurrency; i++ {
		select {
		case result := <-resultsChan:
			resultsMap[result] = true
		default:
			t.Errorf("not enough results in the channel, died at %d", i)
			break
		}
	}

	if len(resultsMap) != concurrency {
		t.Errorf("expected %d results, got %d", concurrency, len(resultsMap))
	}
}

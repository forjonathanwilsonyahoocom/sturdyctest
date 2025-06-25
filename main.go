package main

import (
	"context"
	"github.com/viccon/sturdyc"
	"log"
	"math/rand/v2"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func pickRandomValue(batches [][]string) string {
	batch := batches[rand.IntN(len(batches))]
	return batch[rand.IntN(len(batch))]
}

func demonstrateGetOrFetchBatch(cacheClient *sturdyc.Client[int]) {
	var count atomic.Int32
	fetchFn := func(_ context.Context, ids []string) (map[string]int, error) {
		count.Add(1)
		//log.Printf("we are requesting: %v\n", ids)
		//time.Sleep(time.Millisecond * 1)

		response := make(map[string]int, len(ids))
		for _, id := range ids {
			num, _ := strconv.Atoi(id)
			response[id] = num
		}

		return response, nil
	}
	batchSize := 2000
	numBatches := 100000
	numSubsequentCalls := 100

	batches := make([][]string, numBatches)

	for i := range numBatches {
		batch := make([]string, batchSize)
		for j := range batchSize {
			batch[j] = strconv.Itoa(j + i)
		}
		batches[i] = batch
	}

	// We are going to pass a cache a key function that prefixes each id with
	// the string "my-data-source", and adds an -ID- separator before the actual
	// id. This makes it possible to save the same id for different data
	// sources as the keys would look something like this: my-data-source-ID-1
	keyPrefixFn := cacheClient.BatchKeyFn("my-data-source")

	// Request the keys  for each batch.
	for _, batch := range batches {
		go func() {
			res, _ := cacheClient.GetOrFetchBatch(context.Background(), batch, keyPrefixFn, fetchFn)
			if len(res) < batchSize {
				log.Printf("got batch with unexpected size: %v", len(res))
			}
		}()
	}

	// Give the goroutines above a chance to run to ensure that the batches are in-flight.
	//time.Sleep(time.Second * 3)

	// Launch another 5 goroutines that are going to pick two random IDs from any of the batches.
	var wg sync.WaitGroup
	for i := 0; i < numSubsequentCalls; i++ {
		wg.Add(1)
		go func() {
			ids := []string{pickRandomValue(batches), pickRandomValue(batches), pickRandomValue(batches)}
			res, _ := cacheClient.GetOrFetchBatch(context.Background(), ids, keyPrefixFn, fetchFn)
			if len(res) < 3 {
				log.Printf("subsequently got batch: %v", len(res))
			}
			wg.Done()
		}()
	}

	wg.Wait()
	log.Printf("fetchFn was called %d times\n", count.Load())
}

func main() {

	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	defer f.Close() // error handling omitted for example
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	mf, e := os.Create("mem.prof")
	if e != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	defer mf.Close() // error handling omitted for example
	runtime.GC()     // get up-to-date statistics
	// Lookup("allocs") creates a profile similar to go test -memprofile.
	// Alternatively, use Lookup("heap") for a profile
	// that has inuse_space as the default index.
	if err := pprof.Lookup("allocs").WriteTo(mf, 0); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}

	// Maximum number of entries in the sturdyc.
	capacity := 300000
	// Number of shards to use for the sturdyc.
	numShards := 10
	// Time-to-live for cache entries.
	ttl := 2 * time.Hour
	// Percentage of entries to evict when the cache is full. Setting this
	// to 0 will make set a no-op if the cache has reached its capacity.
	evictionPercentage := 10

	// Create a cache client with the specified configuration.
	cacheClient := sturdyc.New[int](capacity, numShards, ttl, evictionPercentage)

	demonstrateGetOrFetchBatch(cacheClient)
}

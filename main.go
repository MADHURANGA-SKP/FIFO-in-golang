package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// fanIn merges multiple input channels into a single output channel
func fanIn[T any](done <-chan int, channels ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	fannedInStream := make(chan T)
	// transfer copies data from an input channel to the fannedInStream channel
	transfer := func(c <-chan T){
		defer wg.Done()
		for i := range c {
			select {
			case <-done:
				return
			case fannedInStream <- i:
			}
		}
	}
	// Start a goroutine for each input channel to transfer its data
	for _, c := range channels {
		wg.Add(1)
		go transfer(c)
	}
	// Close the output channel once all input channels are processed
	go func ()  {
		wg.Wait()
		close(fannedInStream)
	}()

	return fannedInStream
}

//in this function T any means it uses genereics
//it doesn't limit the funtions that pass into this
func repeatFunc[T any, K any](done <-chan K, fn func() T) <-chan T {
	stream := make(chan T)
	go func ()  {
		defer close(stream)
		for {
			select {
			case <-done:
				return
			case stream <- fn():
			}
		}
	} ()

	return stream
}
// take reads a specified number of values from a channel and sends them
func take[T any, K any](done <-chan K, stream <-chan T, n int) <-chan T {
	taken := make(chan T)
	go func ()  {
		defer close(taken)
		for i := 0; i < n; i++{
			select {
			case <-done:
				return
				//writing the value to taken
			case taken <- <-stream:
			}
		}
	} ()

	return taken
}
// primeFinder reads integers from a channel, checks if they are prime, and sends primes to another channel
func primeFinder(done <-chan int, randIntStream <-chan int) <-chan int {
	isPrime := func(randomInt int) bool {
		for i := randomInt - 1; i > 1; i-- {
			if randomInt%i == 0 {
				return false
			}
		}
		return true
	}

	primes := make(chan int)
	go func ()  {
		defer close(primes)
		for {
			select {
			case <- done:
				return
			case randomInt := <- randIntStream:
				if isPrime(randomInt){
					primes <- randomInt
				}
			}
		}
	}()

	return primes
}

func main() {
	start := time.Now()

	done := make(chan int)
	defer close(done)

	randNumFetcher := func() int{return rand.Intn(500000000)}
	randIntStream := repeatFunc(done, randNumFetcher)
	// primeStrean := primeFinder(done, randIntStream)
	// for rando := range take(done, primeStrean, 10) {// repeatFunc(done, randNumFetcher) {
	// 	fmt.Println(rando)
	// }

	//fan out , start multiple go routines to find prime numbers concurrently
	CPUcount := runtime.NumCPU()
	primeFinderChannels := make([]<-chan int, CPUcount)
	for i := 0; i < CPUcount; i++ {
		primeFinderChannels[i] = primeFinder(done, randIntStream)
	}

	//fan in , merge the result of all prime finders too single channel
	fanInStream := fanIn(done, primeFinderChannels...)
	//takes 10 prime numbers from the fan stream and print them
	for rando := range take(done, fanInStream, 10) {// repeatFunc(done, randNumFetcher) {
		fmt.Println(rando)
	}

	fmt.Println(time.Since(start))
}
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

func doBurn(numWorkers int) {
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			for {
				burn_helper(100)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func burn_helper(i int) int {
	if i <= 0 {
		return 0
	}
	return burn_helper(i - 1)
}

func main() {
	var cpuList string
	var burn bool
	var burnCount int
	flag.StringVar(&cpuList, "cpu-list", "0", "Flag passed to taskset")
	flag.BoolVar(&burn, "burn", false, "Burn cpu cycles")
	flag.IntVar(&burnCount, "burn-count", 1, "Number of burn workers")

	flag.Parse()

	if burn {
		doBurn(burnCount)
		return
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	selfBinary := os.Args[0]
	burnCmd := exec.Command("taskset", "-c", cpuList, selfBinary, "-burn", fmt.Sprintf("-burn-count=%d", burnCount))

	targetArgs := []string{"-c", cpuList}
	for _, arg := range flag.Args() {
		targetArgs = append(targetArgs, arg)
	}
	targetCmd := exec.Command("taskset", targetArgs...)
	stdout, err := targetCmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		wg.Add(1)
		if _, err := io.Copy(os.Stdout, stdout); err != nil {
			log.Printf("copy stdout error: %s", err)
		}
		wg.Done()
	}()
	stderr, err := targetCmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		wg.Add(1)
		if _, err := io.Copy(os.Stderr, stderr); err != nil {
			log.Printf("copy stderr error: %s", err)
		}
		wg.Done()
	}()

	defer func() {
		if err := burnCmd.Process.Kill(); err != nil {
			log.Printf("burn kill (%v) error: %s", burnCmd.Args, err)
		}
	}()
	go func() {
		if err := burnCmd.Run(); err != nil {
			log.Printf("burn (%v) error: %s", burnCmd.Args, err)
		}
	}()

	if err := targetCmd.Run(); err != nil {
		log.Printf("target (%v) error: %s", targetCmd.Args, err)
	}
}

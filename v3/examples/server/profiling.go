//go:build profiling
// +build profiling

package main

import (
	"fmt"
	"os"
	"runtime/pprof"
)

var cpuProfile *os.File
var memProfile *os.File
var stopProfile chan byte

func startProfiling() {
	var err error
	if cpuProfile, err = os.Create("server_cpu.prof"); err != nil {
		panic(err)
	}

	if memProfile, err = os.Create("server_mem.prof"); err != nil {
		panic(err)
	}

	if err = pprof.StartCPUProfile(cpuProfile); err != nil {
		panic(err)
	}
	stopProfile = make(chan byte)
	fmt.Println("Profiling memory and heap...")
	/*
		go func() {
			for {
				select {
				case <-stopProfile:
					fmt.Println("memory profiling goroutine stopped.")
					return
				default:
				}
				time.Sleep(1 * time.Second)
				//runtime.GC()
				profileMemory()
			}
		}()
	*/
}

func profileMemory() {
	if memProfile != nil {
		if err := pprof.WriteHeapProfile(memProfile); err != nil {
			panic(err)
		}
	}
}

func stopProfiling() {
	if cpuProfile != nil {
		pprof.StopCPUProfile()
		cpuProfile.Close()
		cpuProfile = nil
		fmt.Println("CPU profiling stopped.")
	}

	//stopProfile <- 0
	if memProfile != nil {
		fmt.Println("Taking memory sample...")
		profileMemory()
		memProfile.Close()
		memProfile = nil
		fmt.Println("Heap profiling stopped.")
	}
}

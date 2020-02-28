package main

import (
	"flag"
	"fmt"

	"os"
	"os/signal"

	"runtime"

	"github.com/rakyll/hey/requester"
)

var (
	c             = flag.Int("c", 50, "")
	dynamicfrom   = flag.Int64("dynamic-from", 0, "")
	dynamicto     = flag.Int64("dynamic-to", 0, "")
	dynamicprefix = flag.String("dynamic-prefix", "", "")
	redisaddress  = flag.String("redis-address", "", "")
	bodysize      = flag.Int64("body-size", 1, "")
	commandformat = flag.String("command-format", "", "")
	cpus          = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")
)

var usage = `Usage: hey [options...] <url>
Options:
  -c  Number of requests to run concurrently. Total number of requests cannot
      be smaller than the concurrency level. Default is 50.
  -cpus                 Number of used cpu cores.
                        (default for current machine is %d cores)

  -dynamic-from
  -dynamic-to
  -dynamic-prefix
  -redis-address
  -body-size
  -command-format
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}

	flag.Parse()

	runtime.GOMAXPROCS(*cpus)
	conc := *c

	dynamicFrom := *dynamicfrom
	dynamicTo := *dynamicto
	dynamicPrefix := *dynamicprefix
	redisAddress := *redisaddress
	bodySize := *bodysize
	commandFormat := *commandformat

	if conc <= 0 {
		usageAndExit("-n and -c cannot be smaller than 1.")
	}

	if dynamicFrom > dynamicTo {
		usageAndExit("-dynamic-from must <= -dynamic-to.")
	}

	w := &requester.Work{
		C:             conc,
		DynamicFrom:   dynamicFrom,
		DynamicTo:     dynamicTo,
		DynamicPrefix: dynamicPrefix,
		RedisAddress:  redisAddress,
		CommandFormat: commandFormat,
		BodySize:      bodySize,
	}
	w.Init()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		w.Stop()
	}()

	w.Run()
}

func errAndExit(msg string) {
	fmt.Fprintf(os.Stderr, msg)
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

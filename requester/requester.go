package requester

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"log"

	"sync"
	"time"
)

// Max size of the buffer of result channel.
const maxResult = 1000000
const maxIdleConn = 500

type Work struct {
	// C is the concurrency level, the number of concurrent workers to run.
	C int

	initOnce sync.Once
	stopCh   chan struct{}

	DynamicFrom   int64
	DynamicTo     int64
	DynamicPrefix string
	RedisAddress  string
	CommandFormat string
	BodySize      int64
	bodyBuffer    bytes.Buffer
}

func (b *Work) Init() {
	b.initOnce.Do(func() {
		b.stopCh = make(chan struct{}, b.C)
		for i := 0; int64(i) < b.BodySize; i++ {
			b.bodyBuffer.WriteString("z")
		}
	})
	log.Printf("[Init] body: %s", b.bodyBuffer.String())
}

func (b *Work) Run() {
	b.Init()
	b.runWorkers()
	b.Finish()
}

func (b *Work) Stop() {
	for i := 0; i < b.C; i++ {
		b.stopCh <- struct{}{}
	}
}

func (b *Work) Finish() {

}

func (b *Work) makeRequest(cli *redis.Client, n int64) {
	key := fmt.Sprintf("key:%012d", n)
	//log.Printf("[makeRequest] n: %012d, key:%s", n, key)
	v, er := cli.Do("SET", key, b.bodyBuffer.String()).Result()
	if er != nil {
		log.Printf("[makeRequest] Do fail, err: %v, return:%s", er, v)
	}
	if n%10000 == 0 {
		log.Printf("[-- makeRequest --] request: %d completed.", n)
	}

}

func (b *Work) runWorker(start, end int64) {
	//log.Printf("[runWorker] from:%d, to:%d", start, end)
	cli, err := connectRedis(&redis.Options{Addr: b.RedisAddress, ReadTimeout: -1, WriteTimeout: -1})
	if err != nil {
		log.Printf("[runWorker] error %v", err)
	}
	defer cli.Close()

	for i := start; i < end; i++ {
		select {
		case <-b.stopCh:
			return
		default:
			b.makeRequest(cli, i)
		}
	}
	log.Printf("[runWorker] completed request from:%d, to:%d", start, end)
}

func (b *Work) runWorkers() {
	var wg sync.WaitGroup
	wg.Add(b.C)
	sum := b.DynamicTo - b.DynamicFrom
	step := sum / int64(b.C)
	stepSum := b.DynamicFrom
	log.Printf("[runWorkers] sum: %d, concurrent:%d, step:%d,stepSum:%d", sum, b.C, step, stepSum)
	for i := 0; i < b.C; i++ {
		go func(start, end int64) {
			b.runWorker(start, end)
			wg.Done()
		}(stepSum, stepSum+step-1)
		stepSum += step
	}
	wg.Wait()
}

func connectRedis(opts *redis.Options) (*redis.Client, error) {
	var cli *redis.Client
	if opts.Addr == "" {
		return nil, errors.New("address empty")
	}
	return cli, withRetry(3, func() error {
		cli = redis.NewClient(opts)
		return cli.Ping().Err()
	})
}

func withRetry(maxTries int, f func() error) error {
	var err error
	sleepTime := 1 * time.Second
	for i := 1; i <= maxTries; i++ {
		if err = f(); err == nil {
			return nil
		}
		time.Sleep(sleepTime)
	}
	return err
}

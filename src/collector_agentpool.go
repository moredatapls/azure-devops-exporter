package main

import (
	"context"
	"sync"
	"time"
)

type CollectorAgentPool struct {
	CollectorBase

	Processor CollectorProcessorAgentPoolInterface
	Name string
	AgentPoolIdList []int64
}

func (c *CollectorAgentPool) Run(scrapeTime time.Duration) {
	c.SetScrapeTime(scrapeTime)

	c.Processor.Setup(c)
	go func() {
		for {
			go func() {
				c.Collect()
			}()
			Logger.Verbose("collector[%s]: sleeping %v", c.Name, c.GetScrapeTime().String())
			time.Sleep(*c.GetScrapeTime())
		}
	}()
}

func (c *CollectorAgentPool) Collect() {
	var wg sync.WaitGroup
	var wgCallback sync.WaitGroup

	ctx := context.Background()

	callbackChannel := make(chan func())

	Logger.Messsage(
		"collector[%s]: starting metrics collection",
		c.Name,
	)

	wg.Add(1)
	go func(ctx context.Context, callback chan<- func()) {
		defer wg.Done()
		c.Processor.Collect(ctx, callbackChannel)
	}(ctx, callbackChannel)

	// collect metrics (callbacks) and proceses them
	wgCallback.Add(1)
	go func() {
		defer wgCallback.Done()
		var callbackList []func()
		for callback := range callbackChannel {
			callbackList = append(callbackList, callback)
		}

		// reset metric values
		c.Processor.Reset()

		// process callbacks (set metrics)
		for _, callback := range callbackList {
			callback()
		}
	}()

	// wait for all funcs
	wg.Wait()
	close(callbackChannel)
	wgCallback.Wait()

	Logger.Verbose(
		"collector[%s]: finished metrics collection",
		c.Name,
	)
}
package objects

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/validator.v2"

	"github.com/cenkalti/backoff"
	"github.com/segmentio/go-tableize"
	"github.com/tj/go-sync/semaphore"
)

const (
	// Version of the client library.
	Version = "0.0.1"

	// Endpoint for Segment Objects API.
	DefaultBaseEndpoint = "https://objects.segment.com"

	// Default source
	DefaultSource = "project"
)

var (
	ErrClientClosed = errors.New("Client is closed")
)

type Config struct {
	BaseEndpoint string
	Logger       *log.Logger
	Client       *http.Client

	Source string

	MaxBatchBytes    int
	MaxBatchCount    int
	MaxBatchInterval time.Duration

	PrintErrors bool
}

type Client struct {
	Config
	writeKey  string
	wg        sync.WaitGroup
	semaphore semaphore.Semaphore
	closed    int64
	cmap      concurrentMap
}

func New(writeKey string) *Client {
	return NewWithConfig(writeKey, Config{})
}

func NewWithConfig(writeKey string, config Config) *Client {
	conf := getFinalConfig(config)
	return &Client{
		Config:    conf,
		writeKey:  writeKey,
		cmap:      newConcurrentMap(),
		semaphore: make(semaphore.Semaphore, 10),
	}
}

func getFinalConfig(c Config) Config {
	if c.BaseEndpoint == "" {
		c.BaseEndpoint = DefaultBaseEndpoint
	}

	if c.Logger == nil {
		c.Logger = log.New(os.Stderr, "segment ", log.LstdFlags)
	}

	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	if c.MaxBatchBytes <= 0 {
		c.MaxBatchBytes = 500 << 10
	}

	if c.MaxBatchCount <= 0 {
		c.MaxBatchCount = 100
	}

	if c.MaxBatchInterval <= 0 {
		c.MaxBatchInterval = 10 * time.Second
	}

	if c.Source == "" {
		c.Source = DefaultSource
	}

	return c
}

func (c *Client) fetchFunction(key string) *buffer {
	b := newBuffer(key)
	c.wg.Add(1)
	go c.buffer(b)
	return b
}

func (c *Client) flush(b *buffer) {
	if b.count() == 0 {
		return
	}

	rm := b.marshalArray()
	c.semaphore.Run(func() {
		batchRequest := &batch{
			Source:     c.Source,
			Collection: b.collection,
			WriteKey:   c.writeKey,
			Objects:    rm,
		}

		err := c.makeRequest(batchRequest)
		if c.PrintErrors {
			log.Printf("[ERROR] Batch failed making request: %v", err)
		}
	})
	b.reset()
}

func (c *Client) buffer(b *buffer) {
	defer c.wg.Done()

	tick := time.NewTicker(c.MaxBatchInterval)
	defer tick.Stop()

	for {
		select {
		case req := <-b.Channel:
			req.Properties = tableize.Tableize(&tableize.Input{
				Value: req.Properties,
			})
			x, err := json.Marshal(req)
			if err != nil {
				if c.PrintErrors {
					log.Printf("[Error] Message `%s` excluded from batch: %v", req.ID, err)
				}
				continue
			}
			if b.size()+len(x) >= c.MaxBatchBytes || b.count()+1 >= c.MaxBatchCount {
				c.flush(b)
			}
			b.add(x)
		case <-tick.C:
			c.flush(b)
		case <-b.Exit:
			for req := range b.Channel {
				req.Properties = tableize.Tableize(&tableize.Input{
					Value: req.Properties,
				})
				x, err := json.Marshal(req)
				if err != nil {
					if c.PrintErrors {
						log.Printf("[Error] Exiting: Message `%s` excluded from batch: %v", req.ID, err)
					}
					continue
				}
				if b.size()+len(x) >= c.MaxBatchBytes || b.count()+1 >= c.MaxBatchCount {
					c.flush(b)
				}
				b.add(x)
			}
			c.flush(b)
			return
		}
	}

}

func (c *Client) Close() error {
	if !atomic.CompareAndSwapInt64(&c.closed, 0, 1) {
		return ErrClientClosed
	}

	for t := range c.cmap.Iter() {
		t.Val.Exit <- struct{}{}
		close(t.Val.Exit)
		close(t.Val.Channel)
	}

	c.wg.Wait()
	c.semaphore.Wait()

	return nil
}

func (c *Client) Set(v *Object) error {
	if atomic.LoadInt64(&c.closed) == 1 {
		return ErrClientClosed
	}

	if err := validator.Validate(v); err != nil {
		return err
	}

	c.cmap.Fetch(v.Collection, c.fetchFunction).Channel <- v
	return nil
}

func (c *Client) makeRequest(request *batch) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 10 * time.Second
	err = backoff.Retry(func() error {
		bodyReader := bytes.NewReader(payload)
		resp, err := http.Post(c.BaseEndpoint+"/v1/set", "application/json", bodyReader)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		response := map[string]interface{}{}
		dec := json.NewDecoder(resp.Body)
		dec.Decode(&response)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP Post Request Failed, Status Code %d. \nResponse: %v",
				resp.StatusCode, response)
		}

		return nil
	}, b)

	return err
}

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
)

var (
	ErrClientClosed = errors.New("Client is closed")
)

type Client struct {
	BaseEndpoint string
	Logger       *log.Logger
	Client       *http.Client

	MaxBatchBytes    int
	MaxBatchCount    int
	MaxBatchInterval time.Duration

	writeKey  string
	wg        sync.WaitGroup
	semaphore semaphore.Semaphore
	closed    int64
	cmap      concurrentMap
}

func New(writeKey string) *Client {
	return &Client{
		BaseEndpoint:     DefaultBaseEndpoint,
		Logger:           log.New(os.Stderr, "segment ", log.LstdFlags),
		writeKey:         writeKey,
		Client:           http.DefaultClient,
		cmap:             newConcurrentMap(),
		MaxBatchBytes:    500 << 10,
		MaxBatchCount:    100,
		MaxBatchInterval: 10 * time.Second,
		semaphore:        make(semaphore.Semaphore, 10),
	}
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
			Collection: b.collection,
			WriteKey:   c.writeKey,
			Objects:    rm,
		}

		c.makeRequest(batchRequest)
	})
	b.reset()
}

func (c *Client) buffer(b *buffer) {
	defer c.wg.Done()

	tick := time.NewTicker(c.MaxBatchInterval)

	for {
		select {
		case req := <-b.Channel:
			req.Properties = tableize.Tableize(&tableize.Input{
				Value: req.Properties,
			})
			x, err := json.Marshal(req)
			if err != nil {
				log.Printf("[Error] Message `%s` excluded from batch: %v", req.ID, err)
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
					log.Printf("[Error] Message `%s` excluded from batch: %v", req.ID, err)
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

func (c *Client) makeRequest(request *batch) {
	payload, err := json.Marshal(request)
	if err != nil {
		log.Printf("[Error] Batch failed to marshal: %v - %v", request, err)
		return
	}

	bodyReader := bytes.NewReader(payload)

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 10 * time.Second
	err = backoff.Retry(func() error {
		resp, err := http.Post(c.BaseEndpoint+"/v1/set", "application/json", bodyReader)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		response := map[string]interface{}{}
		dec := json.NewDecoder(resp.Body)
		dec.Decode(&response)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP Post Request Failed, Status Code %d: %v", resp.StatusCode, response)
		}

		return nil
	}, b)

	if err != nil {
		log.Printf("[Error] %v", err)
		return
	}
}

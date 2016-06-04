package objects

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

func TestClient(t *testing.T) {
	suite.Run(t, &ClientTestSuite{})
}

type ClientTestSuite struct {
	suite.Suite

	httpRequestsMutex sync.Mutex
	httpRequests      []*batch
	httpSuccess       int64
	httpErrors        int64
}

func (c *ClientTestSuite) SetupSuite() {
	httpmock.Activate()

	responder := func(req *http.Request) (*http.Response, error) {
		defer req.Body.Close()

		v := &batch{}
		dec := json.NewDecoder(req.Body)

		if err := dec.Decode(v); err != nil {
			atomic.AddInt64(&c.httpErrors, 1)
			return httpmock.NewStringResponse(500, ""), nil
		}

		c.httpRequestsMutex.Lock()
		c.httpRequests = append(c.httpRequests, v)
		c.httpRequestsMutex.Unlock()
		atomic.AddInt64(&c.httpSuccess, 1)

		return httpmock.NewStringResponse(200, `{"success": true}`), nil
	}

	httpmock.RegisterResponder("POST", "https://objects.segment.com/v1/set", responder)
}

func (c *ClientTestSuite) TestNewClient() {
	client := New("writeKey")
	c.NotNil(client)
	c.NotEmpty(client.BaseEndpoint)
	c.NotNil(client.Client)
	c.NotNil(client.Logger)
	c.NotNil(client.semaphore)
	c.NotNil(client.wg)
	c.Equal("writeKey", client.writeKey)
	c.Equal(0, client.cmap.Count())
}

func (c *ClientTestSuite) TestSetOnce() {
	client := New("writeKey")
	c.NotNil(client)

	v := &Object{ID: "id", Collection: "c", Properties: map[string]interface{}{"p": "1"}}
	c.NoError(client.Set(v))
	c.Equal(1, client.cmap.Count())
}

func (c *ClientTestSuite) TestSetFull() {
	client := New("writeKey")
	c.NotNil(client)

	v1 := &Object{ID: "id", Collection: "c", Properties: map[string]interface{}{"p": "1"}}
	c.NoError(client.Set(v1))
	v2 := &Object{ID: "id2", Collection: "c", Properties: map[string]interface{}{"p": "2"}}
	c.NoError(client.Set(v2))

	c.Equal(1, client.cmap.Count())

	client.Close()

	c.Len(c.httpRequests, 1)
	c.Equal("c", c.httpRequests[0].Collection)

	received := []*Object{}
	c.NoError(json.Unmarshal(c.httpRequests[0].Objects, &received))
	c.Len(received, 2)
	c.Equal("id", received[0].ID)
	c.Equal("id2", received[1].ID)
}

func (c *ClientTestSuite) TestChannelFlow() {
	client := New("writeKey")
	c.NotNil(client)

	v := &Object{ID: "id", Collection: "c", Properties: map[string]interface{}{"p": "1"}}

	buf := client.cmap.Fetch("c", client.fetchFunction)
	buf.Channel <- v

	// TODO(vince): Find a better solution to test this
	// Wait for the channel to add to buffer
	time.Sleep(250 * time.Millisecond)
	c.Equal(1, buf.count())

	bt, err := json.Marshal(v)
	c.NoError(err)

	c.Equal(len(bt), buf.size())
}

func (c *ClientTestSuite) TestSetErrors() {
	client := New("writeKey")
	c.NotNil(client)

	// Error with empty object
	c.Error(client.Set(&Object{}))

	// Error with empty Collection
	c.Error(client.Set(&Object{ID: "id", Collection: "", Properties: map[string]interface{}{"prop1": "1"}}))

	// Error without properties
	c.Error(client.Set(&Object{ID: "id", Collection: "collection"}))

	// Error with empty ID
	c.Error(client.Set(&Object{ID: "", Collection: "collection", Properties: map[string]interface{}{"prop1": "1"}}))
}

func (c *ClientTestSuite) TestClose() {
	client := New("writeKey")
	c.NotNil(client)
	client.Close()

	// Error already closed
	c.Error(client.Set(&Object{ID: "id", Collection: "collection", Properties: map[string]interface{}{"prop1": "1"}}))
}

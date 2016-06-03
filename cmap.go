package objects

import (
	"encoding/json"
	"hash/fnv"
	"sync"
)

const shardCount = 32

// A "thread" safe map of type string:*buffer.
// To avoid lock bottlenecks this map is dived to several (shardCount) map shards.
type concurrentMap []*concurrentMapShared
type concurrentMapShared struct {
	items        map[string]*buffer
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// Creates a new concurrent map.
func NewConcurrentMap() concurrentMap {
	m := make(concurrentMap, shardCount)
	for i := 0; i < shardCount; i++ {
		m[i] = &concurrentMapShared{items: make(map[string]*buffer)}
	}
	return m
}

// Returns shard under given key
func (m concurrentMap) GetShard(key string) *concurrentMapShared {
	hasher := fnv.New32()
	hasher.Write([]byte(key))
	return m[int(hasher.Sum32())%shardCount]
}

// Sets the given value under the specified key.
func (m *concurrentMap) Set(key string, value *buffer) {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	shard.items[key] = value
}

// Retrieves an element from map under given key.
func (m concurrentMap) Get(key string) (*buffer, bool) {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	defer shard.RUnlock()

	// Get item from shard.
	val, ok := shard.items[key]
	return val, ok
}

// Sets the given value under the specified key.
func (m *concurrentMap) Fetch(key string, f func(key string) *buffer) (v *buffer) {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	v, ok := shard.items[key]
	if !ok {
		v = f(key)
		shard.items[key] = v
	}

	return
}

// Returns the number of elements within the map.
func (m concurrentMap) Count() int {
	count := 0
	for i := 0; i < shardCount; i++ {
		shard := m[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Looks up an item under specified key
func (m *concurrentMap) Has(key string) bool {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	defer shard.RUnlock()

	// See if element is within shard.
	_, ok := shard.items[key]
	return ok
}

// Removes an element from the map.
func (m *concurrentMap) Remove(key string) {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	delete(shard.items, key)
}

// Checks if map is empty.
func (m *concurrentMap) IsEmpty() bool {
	return m.Count() == 0
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type Tuple struct {
	Key string
	Val *buffer
}

// Returns an iterator which could be used in a for range loop.
func (m concurrentMap) Iter() <-chan Tuple {
	ch := make(chan Tuple)
	go func() {
		// Foreach shard.
		for _, shard := range m {
			// Foreach key, value pair.
			shard.RLock()
			for key, val := range shard.items {
				ch <- Tuple{key, val}
			}
			shard.RUnlock()
		}
		close(ch)
	}()
	return ch
}

// Returns a buffered iterator which could be used in a for range loop.
func (m concurrentMap) IterBuffered() <-chan Tuple {
	ch := make(chan Tuple, m.Count())
	go func() {
		// Foreach shard.
		for _, shard := range m {
			// Foreach key, value pair.
			shard.RLock()
			for key, val := range shard.items {
				ch <- Tuple{key, val}
			}
			shard.RUnlock()
		}
		close(ch)
	}()
	return ch
}

//Reviles concurrentMap "private" variables to json marshal.
func (m concurrentMap) MarshalJSON() ([]byte, error) {
	// Create a temporary map, which will hold all item spread across shards.
	tmp := make(map[string]*buffer)

	// Insert items to temporary map.
	for item := range m.Iter() {
		tmp[item.Key] = item.Val
	}
	return json.Marshal(tmp)
}

func (m *concurrentMap) UnmarshalJSON(b []byte) (err error) {
	// Reverse process of Marshal.

	tmp := make(map[string]*buffer)

	// Unmarshal into a single map.
	if err := json.Unmarshal(b, &tmp); err != nil {
		return nil
	}

	// foreach key,value pair in temporary map insert into our concurrent map.
	for key, val := range tmp {
		m.Set(key, val)
	}
	return nil
}

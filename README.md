# objects-go

  Segment objects client for Go. For additional documentation
  visit [https://segment.com/docs/libraries/go](https://segment.com/docs/libraries/go/) or view the [godocs](http://godoc.org/github.com/segmentio/objects-go).

## Description
Segment’s Objects API allows you to send stateful business objects right to Redshift and other Segment supported data warehouses. These objects can be anything that is relevant to your business: products in your product catalogs, partners on your platform, articles on your blog, etc.

The Objects API lets you `set` custom objects in your own data warehouse.

```go
// First call to Set
Client.Set(*objects.Object{
  ID: "room1000",
  Collection: "rooms"
  Properties: map[string]interface{}{
    "name": "Charming Beach Room Facing Ocean",
    "location": "Lihue, HI",
    "review_count": 47,
})

// Second call on the same object 
Client.Set(*objects.Object{
  ID: "room1000",
  Collection: "rooms"
  Properties: map[string]interface{}{
    "owner": "Calvin",
    "public_listing": true,
})
```

This call makes the objects available in your data warehouse…

```SQL
select id, name, location, review_count, owner, public_listing from hotel.rooms
```

..which will return…

```CSV
'room1000' | 'Charming Beach Room Facing Ocean' | 'Lihue, HI' | 47 | "Calvin" | true
```

> All objects will be flattened using the `go-tableize` library. Objects API doesn't allow nested objects, empty objects, and only allows strings, numeric types or booleans as values.

## HTTP API 

There is a single `.set` HTTP API endpoint that you'll use to send data to Segment. 


    POST https://objects.segment.com/v1/set

with the following payload: 


    {
      "collection": "rooms",
      "objects": [
        {
          "id": "2561341",
          "properties": {
            "name": "Charming Beach Room Facing Ocean",
            "location": "Lihue, HI",
            "review_count": 47
          }
        }, {
          "id": "2561342",
          "properties": {
            "name": "College town feel — plenty of bars nearby",
            "location": "Austin, TX",
            "review_count": 32
          }
        }
      ]
    }

Here’s a `curl` example of how to get started: 


    curl https://objects.segment.com/v1/set \
       -u PROJECT_WRITE_KEY: \
       -H 'Content-Type: application/json' \
       -X POST -d '{"collection":"rooms","objects":[{"id": "2561341","properties": {"name": "Charming Beach Room Facing Ocean","location":"Lihue, HI","review_count":47}}]}'


## License

 MIT

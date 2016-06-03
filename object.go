package objects

type Object struct {
	Collection string                 `json:"-"`
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
}

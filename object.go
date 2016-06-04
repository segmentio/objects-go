package objects

type Object struct {
	Collection string                 `json:"-" validate:"nonzero"`
	ID         string                 `json:"id" validate:"nonzero"`
	Properties map[string]interface{} `json:"properties" validate:"min=1"`
}

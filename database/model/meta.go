package model

type Meta struct {
	Created   int64 `json:"c,omitempty" bson:"c,omitempty"`
	Modified  int64 `json:"m,omitempty" bson:"m,omitempty"`
	Expires   int64 `json:"e,omitempty" bson:"e,omitempty"`
	Deleted   int64 `json:"d,omitempty" bson:"d,omitempty"`
	Secret    bool  `json:"s,omitempty" bson:"s,omitempty"` // secrets must not be sent to the UI, only synced between nodes
	Cronjewel bool  `json:"j,omitempty" bson:"j,omitempty"` // crownjewels must never leave the instance, but may be read by the UI
}

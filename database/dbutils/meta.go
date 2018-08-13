// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package dbutils

type Meta struct {
	Created   int64 `json:"c,omitempty" bson:"c,omitempty"`
	Modified  int64 `json:"m,omitempty" bson:"m,omitempty"`
	Expires   int64 `json:"e,omitempty" bson:"e,omitempty"`
	Deleted   int64 `json:"d,omitempty" bson:"d,omitempty"`
	Secret    bool  `json:"s,omitempty" bson:"s,omitempty"` // secrets must not be sent to clients, only synced between cores
	Cronjewel bool  `json:"j,omitempty" bson:"j,omitempty"` // crownjewels must never leave the instance
}

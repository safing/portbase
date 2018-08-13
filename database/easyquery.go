// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"errors"
	"fmt"
	"strings"

	dsq "github.com/ipfs/go-datastore/query"
)

type FilterMaxDepth struct {
	MaxDepth int
}

func (f FilterMaxDepth) Filter(entry dsq.Entry) bool {
	return strings.Count(entry.Key, "/") <= f.MaxDepth
}

type FilterKeyLength struct {
	Length int
}

func (f FilterKeyLength) Filter(entry dsq.Entry) bool {
	return len(entry.Key) == f.Length
}

func EasyQueryIterator(subscriptionKey string) (dsq.Results, error) {
	query := dsq.Query{}

	namespaces := strings.Split(subscriptionKey, "/")[1:]
	lastSpace := ""
	if len(namespaces) != 0 {
		lastSpace = namespaces[len(namespaces)-1]
	}

	switch {
	case lastSpace == "":
		// get all children
		query.Prefix = subscriptionKey
	case strings.HasPrefix(lastSpace, "*"):
		// get children to defined depth
		query.Prefix = strings.Trim(subscriptionKey, "*")
		query.Filters = []dsq.Filter{
			FilterMaxDepth{len(lastSpace) + len(namespaces) - 1},
		}
	case strings.Contains(lastSpace, ":"):
		query.Prefix = subscriptionKey
		query.Filters = []dsq.Filter{
			FilterKeyLength{len(query.Prefix)},
		}
	default:
		// get only from this location and this type
		query.Prefix = subscriptionKey + ":"
		query.Filters = []dsq.Filter{
			FilterMaxDepth{len(namespaces)},
		}
	}

	// log.Tracef("easyquery: %s has prefix %s", subscriptionKey, query.Prefix)

	results, err := db.Query(query)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("easyquery: %s", err))
	}

	return results, nil
}

func EasyQuery(subscriptionKey string) (*[]dsq.Entry, error) {

	results, err := EasyQueryIterator(subscriptionKey)
	if err != nil {
		return nil, err
	}

	entries, err := results.Rest()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("easyquery: %s", err))
	}

	return &entries, nil
}

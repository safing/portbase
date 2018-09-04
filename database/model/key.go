package model

import (
	"errors"
	"strings"
)

func ParseKey(key string) (dbName, dbKey string, err error) {
	splitted := strings.SplitN(key, ":", 2)
	if len(splitted) == 2 {
		return splitted[0], splitted[1], nil
	}
	return "", "", errors.New("invalid key")
}

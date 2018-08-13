// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"time"

	"github.com/Safing/safing-core/formats/dsd"
	"github.com/Safing/safing-core/log"

	dsq "github.com/ipfs/go-datastore/query"
)

func init() {
	// go dumper()
}

func dumper() {
	for {
		time.Sleep(10 * time.Second)
		result, err := db.Query(dsq.Query{Prefix: "/Run/Process"})
		if err != nil {
			log.Warningf("Query failed: %s", err)
			continue
		}
		log.Infof("Dumping all processes:")
		for model, ok := result.NextSync(); ok; model, ok = result.NextSync() {
			bytes, err := dsd.Dump(model, dsd.AUTO)
			if err != nil {
				log.Warningf("Error dumping: %s", err)
				continue
			}
			log.Info(string(bytes))
		}
		log.Infof("END")
	}
}

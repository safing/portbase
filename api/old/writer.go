// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package api

import (
	"fmt"

	"github.com/Safing/safing-core/database"
	"github.com/Safing/safing-core/formats/dsd"
	"github.com/Safing/safing-core/log"

	"github.com/gorilla/websocket"
	"github.com/ipfs/go-datastore"
)

// Writer writes messages to the client.
func (m *Session) Writer() {

	wsConn := m.wsConn
	defer wsConn.Close()
	sub := m.subscription

	var model database.Model
	var key *datastore.Key
	var msg []byte
	msgCreated := true
	var err error

writeLoop:
	for {

		model = nil
		key = nil
		msg = nil

		select {
		// prioritize direct writes
		case msg = <-m.send:
		default:
			select {
			case msg = <-m.send:
			case model = <-sub.Created:
				msgCreated = true
				// log.Tracef("api: got new from subscription")
			case model = <-sub.Updated:
				msgCreated = false
				// log.Tracef("api: got update from subscription")
			case key = <-sub.Deleted:
				// log.Tracef("api: got delete from subscription")
			}
		}

		if model != nil {
			data, err := database.DumpModel(model, dsd.JSON)
			if err != nil {
				log.Warningf("api: could not dump model: %s", err)
				continue writeLoop
			}
			if msgCreated {
				toSend := append([]byte(fmt.Sprintf("created|%s|", model.GetKey().String())), data...)
				msg = toSend
			} else {
				toSend := append([]byte(fmt.Sprintf("updated|%s|", model.GetKey().String())), data...)
				msg = toSend
			}
		} else if key != nil {
			toSend := append([]byte(fmt.Sprintf("deleted|%s", key.String())))
			msg = toSend
		}

		// exit if we got nil
		if msg == nil {
			log.Debugf("api: a sending channel was closed, stopping writer")
			return
		}

		// log.Tracef("api: sending %s", string(*msg))
		err = wsConn.WriteMessage(websocket.BinaryMessage, msg)
		if err != nil {
			// if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			log.Warningf("api: write error: %s", err)
			// }
			return
		}

	}

}

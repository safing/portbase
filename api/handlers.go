// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package api

import (
	"bytes"
	"fmt"

	"github.com/Safing/safing-core/log"

	"net/http"

	"github.com/gorilla/websocket"
)

func allowAnyOrigin(r *http.Request) bool {
	return true
}

func apiVersionOneHandler(w http.ResponseWriter, r *http.Request) {

	upgrader := websocket.Upgrader{
		CheckOrigin:     allowAnyOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 65536,
	}
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("upgrade to websocket failed: %s\n", err)
		return
	}

	// new or resume session?

	var session *Session

	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		wsConn.Close()
		return
	}

	parts := bytes.SplitN(msg, []byte("|"), 2)
	switch string(parts[0]) {
	case "start":
		session = NewSession(wsConn)
	case "resume":
		if len(parts) > 1 {
			session, err = ResumeSession(string(parts[1]), wsConn)
			if err != nil {
				handleError(session, fmt.Sprintf("error|500|created new session, restoring failed: %s", err))
			} else {
			}
		} else {
			session = NewSession(wsConn)
		}
	default:
		wsConn.Close()
		return
	}

	defer session.Deactivate()

	// start handling requests
	for {

		_, msg, err := wsConn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Warningf("api: read error: %s", err)
			}
			return
		}

		log.Tracef("api: got request %s", string(msg))

		splitParams := bytes.SplitN(msg, []byte("|"), 3)

		if len(splitParams) < 2 {
			handleError(session, "error|400|too few params")
		}

		action, key := string(splitParams[0]), string(splitParams[1])

		// if len(splitParams) > 2 {
		// 	json := splitParams[2]
		// 	log.Infof("JSON: %q", json)
		// }

		switch action {
		case "get":
			Get(session, key)
		case "subscribe":
			Subscribe(session, key)
		case "unsubscribe":
			Unsubscribe(session, key)
		case "create":
			if len(splitParams) < 3 {
				handleError(session, "error|400|invalid action: cannot create without data")
			}
			Save(session, key, true, splitParams[2])
		case "update":
			if len(splitParams) < 3 {
				handleError(session, "error|400|invalid action: cannot update without data")
			}
			Save(session, key, false, splitParams[2])
		case "delete":
			Delete(session, key)
		default:
			handleError(session, "error|400|invalid action: "+action)
		}
	}

}

func handleError(session *Session, message string) {
	log.Warningf("api: " + message)
	toSend := []byte(message)
	session.send <- toSend
}

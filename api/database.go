package api

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/tevino/abool"

	"github.com/Safing/portbase/container"
	"github.com/Safing/portbase/database"
	"github.com/Safing/portbase/database/query"
	"github.com/Safing/portbase/database/record"
	"github.com/Safing/portbase/log"
)

const (
	dbMsgTypeOk      = "ok"
	dbMsgTypeError   = "error"
	dbMsgTypeDone    = "done"
	dbMsgTypeSuccess = "success"
	dbMsgTypeUpd     = "upd"
	dbMsgTypeNew     = "new"
	dbMsgTypeDelete  = "delete"
	dbMsgTypeWarning = "warning"
)

// DatabaseAPI is a database API instance.
type DatabaseAPI struct {
	conn      *websocket.Conn
	sendQueue chan []byte
	subs      map[string]*database.Subscription

	shutdownSignal chan struct{}
	shuttingDown   *abool.AtomicBool
	db             *database.Interface
}

func allowAnyOrigin(r *http.Request) bool {
	return true
}

func startDatabaseAPI(w http.ResponseWriter, r *http.Request) {

	upgrader := websocket.Upgrader{
		CheckOrigin:     allowAnyOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 65536,
	}
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		errMsg := fmt.Sprintf("could not upgrade to websocket: %s", err)
		log.Error(errMsg)
		http.Error(w, errMsg, 400)
		return
	}

	new := &DatabaseAPI{
		conn:           wsConn,
		sendQueue:      make(chan []byte, 100),
		subs:           make(map[string]*database.Subscription),
		shutdownSignal: make(chan struct{}),
		shuttingDown:   abool.NewBool(false),
		db:             database.NewInterface(nil),
	}

	go new.handler()
	go new.writer()
}

func (api *DatabaseAPI) handler() {

	// 123|get|<key>
	//    123|ok|<key>|<data>
	//    123|error|<message>
	// 124|query|<query>
	//    124|ok|<key>|<data>
	//    124|done
	//    124|error|<message>
	// 125|sub|<query>
	//    125|upd|<key>|<data>
	//    125|new|<key>|<data>
	//    125|delete|<key>|<data>
	//    125|warning|<message> // does not cancel the subscription
	// 127|qsub|<query>
	//    127|ok|<key>|<data>
	//    127|done
	//    127|error|<message>
	//    127|upd|<key>|<data>
	//    127|new|<key>|<data>
	//    127|delete|<key>|<data>
	//    127|warning|<message> // does not cancel the subscription

	// 128|create|<key>|<data>
	//    128|success
	//    128|error|<message>
	// 129|update|<key>|<data>
	//    129|success
	//    129|error|<message>
	// 130|insert|<key>|<data>
	//    130|success
	//    130|error|<message>

	for {

		_, msg, err := api.conn.ReadMessage()
		if err != nil {
			if !api.shuttingDown.IsSet() {
				api.shutdown()
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Warningf("api: websocket write error: %s", err)
				}
			}
			return
		}

		parts := bytes.SplitN(msg, []byte("|"), 2)
		if len(parts) != 3 {
			api.send(nil, dbMsgTypeError, []byte("bad request: malformed message"))
			continue
		}

		switch string(parts[1]) {
		case "get":
			// 123|get|<key>
			go api.handleGet(parts[0], string(parts[2]))
		case "query":
			// 124|query|<query>
			go api.handleQuery(parts[0], string(parts[2]))
		case "sub":
			// 125|sub|<query>
			go api.handleSub(parts[0], string(parts[2]))
		case "qsub":
			// 127|qsub|<query>
			go api.handleQsub(parts[0], string(parts[2]))
		case "create", "update", "insert":

			// split key and payload
			dataParts := bytes.SplitN(parts[2], []byte("|"), 1)
			if len(dataParts) != 2 {
				api.send(nil, dbMsgTypeError, []byte("bad request: malformed message"))
				continue
			}

			switch string(parts[1]) {
			case "create":
				// 128|create|<key>|<data>
				go api.handleCreate(parts[0], string(dataParts[0]), dataParts[1])
			case "update":
				// 129|update|<key>|<data>
				go api.handleUpdate(parts[0], string(dataParts[0]), dataParts[1])
			case "insert":
				// 130|insert|<key>|<data>
				go api.handleInsert(parts[0], string(dataParts[0]), dataParts[1])
			}

		default:
			api.send(parts[0], dbMsgTypeError, []byte("bad request: unknown method"))
		}
	}
}

func (api *DatabaseAPI) writer() {
	var data []byte
	var err error

	for {
		data = nil

		select {
		// prioritize direct writes
		case data = <-api.sendQueue:
			if data == nil || len(data) == 0 {
				api.shutdown()
				return
			}
		case <-api.shutdownSignal:
			return
		}

		// log.Tracef("api: sending %s", string(*msg))
		err = api.conn.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			if !api.shuttingDown.IsSet() {
				api.shutdown()
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Warningf("api: websocket write error: %s", err)
				}
			}
			return
		}

	}
}

func (api *DatabaseAPI) send(opID []byte, msgType string, data []byte) {
	c := container.New(opID)
	c.Append([]byte(fmt.Sprintf("|%s|", msgType)))
	c.Append(data)
	api.sendQueue <- c.CompileData()
}

func (api *DatabaseAPI) handleGet(opID []byte, key string) {
	// 123|get|<key>
	//    123|ok|<key>|<data>
	//    123|error|<message>

	var data []byte

	r, err := api.db.Get(key)
	if err == nil {
		data, err = r.Marshal(r, record.JSON)
	} else {
		api.send(opID, dbMsgTypeError, []byte(err.Error()))
		return
	}
	api.send(opID, dbMsgTypeOk, data)
}

func (api *DatabaseAPI) handleQuery(opID []byte, queryText string) {
	// 124|query|<query>
	//    124|ok|<key>|<data>
	//    124|done
	//    124|warning|<message>
	//    124|error|<message>

	var err error

	q, err := query.ParseQuery(queryText)
	if err != nil {
		api.send(opID, dbMsgTypeError, []byte(err.Error()))
		return
	}

	api.processQuery(opID, q)
}

func (api *DatabaseAPI) processQuery(opID []byte, q *query.Query) (ok bool) {
	it, err := api.db.Query(q)
	if err != nil {
		api.send(opID, dbMsgTypeError, []byte(err.Error()))
		return false
	}

	for r := range it.Next {
		data, err := r.Marshal(r, record.JSON)
		if err != nil {
			api.send(opID, dbMsgTypeWarning, []byte(err.Error()))
		}
		api.send(opID, dbMsgTypeOk, data)
	}
	if it.Error != nil {
		api.send(opID, dbMsgTypeError, []byte(err.Error()))
		return false
	}

	api.send(opID, dbMsgTypeDone, nil)
	return true
}

// func (api *DatabaseAPI) runQuery()

func (api *DatabaseAPI) handleSub(opID []byte, queryText string) {
	// 125|sub|<query>
	//    125|upd|<key>|<data>
	//    125|new|<key>|<data>
	//    125|delete|<key>
	//    125|warning|<message> // does not cancel the subscription
	var err error

	q, err := query.ParseQuery(queryText)
	if err != nil {
		api.send(opID, dbMsgTypeError, []byte(err.Error()))
		return
	}

	sub, ok := api.registerSub(opID, q)
	if !ok {
		return
	}
	api.processSub(opID, sub)
}

func (api *DatabaseAPI) registerSub(opID []byte, q *query.Query) (sub *database.Subscription, ok bool) {
	var err error
	sub, err = api.db.Subscribe(q)
	if err != nil {
		api.send(opID, dbMsgTypeError, []byte(err.Error()))
		return nil, false
	}
	return sub, true
}

func (api *DatabaseAPI) processSub(opID []byte, sub *database.Subscription) {
	for r := range sub.Feed {
		data, err := r.Marshal(r, record.JSON)
		if err != nil {
			api.send(opID, dbMsgTypeWarning, []byte(err.Error()))
		}
		// TODO: use upd, new and delete msgTypes
		api.send(opID, dbMsgTypeOk, data)
	}
	if sub.Err != nil {
		api.send(opID, dbMsgTypeError, []byte(sub.Err.Error()))
	}
}

func (api *DatabaseAPI) handleQsub(opID []byte, queryText string) {
	// 127|qsub|<query>
	//    127|ok|<key>|<data>
	//    127|done
	//    127|error|<message>
	//    127|upd|<key>|<data>
	//    127|new|<key>|<data>
	//    127|delete|<key>
	//    127|warning|<message> // does not cancel the subscription

	var err error

	q, err := query.ParseQuery(queryText)
	if err != nil {
		api.send(opID, dbMsgTypeError, []byte(err.Error()))
		return
	}

	sub, ok := api.registerSub(opID, q)
	if !ok {
		return
	}
	ok = api.processQuery(opID, q)
	if !ok {
		return
	}
	api.processSub(opID, sub)
}

func (api *DatabaseAPI) handleCreate(opID []byte, key string, data []byte) {
	// 128|create|<key>|<data>
	//    128|success
	//    128|error|<message>
}
func (api *DatabaseAPI) handleUpdate(opID []byte, key string, data []byte) {
	// 129|update|<key>|<data>
	//    129|success
	//    129|error|<message>
}
func (api *DatabaseAPI) handleInsert(opID []byte, key string, data []byte) {
	// 130|insert|<key>|<data>
	//    130|success
	//    130|error|<message>
}

func (api *DatabaseAPI) shutdown() {
	if api.shuttingDown.SetToIf(false, true) {
		close(api.shutdownSignal)
		api.conn.Close()
	}
}

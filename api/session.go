// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ipfs/go-datastore"
	uuid "github.com/satori/go.uuid"

	"github.com/Safing/safing-core/database"
	"github.com/Safing/safing-core/log"
)

// Session holds data for an api session.
type Session struct {
	database.Base
	ID            string
	wsConn        *websocket.Conn
	Expires       int64
	Subscriptions []string
	subscription  *database.Subscription
	send          chan []byte
}

var sessionModel *Session // only use this as parameter for database.EnsureModel-like functions

func init() {
	database.RegisterModel(sessionModel, func() database.Model { return new(Session) })
}

// NewSession creates a new session.
func NewSession(wsConn *websocket.Conn) *Session {
	session := &Session{
		ID:           strings.Replace(uuid.NewV4().String(), "-", "", -1),
		subscription: database.NewSubscription(),
		send:         make(chan []byte, 1024),
	}
	session.wsConn = wsConn
	session.CreateWithID()
	log.Tracef("api: created new session: %s", session.ID)
	toSend := []byte("session|" + session.ID)
	session.send <- toSend
	go session.Writer()
	return session
}

// ResumeSession an existing session.
func ResumeSession(id string, wsConn *websocket.Conn) (*Session, error) {
	session, err := GetSession(id)
	if err == nil {
		if session.wsConn != nil {
			session.wsConn.Close()
		}
		session.wsConn = wsConn
		session.Save()
		log.Tracef("api: resumed session %s", session.ID)
		go session.Writer()
		return session, nil
	}
	return NewSession(wsConn), fmt.Errorf("api: failed to restore session %s, creating new", id)
}

// Deactivate closes down a session, making it ready to be resumed.
func (m *Session) Deactivate() {
	m.wsConn.Close()
	m.wsConn = nil
	m.subscription.Destroy()
	m.subscription = nil
	m.Save()
}

// Subscribe subscribes to a database key and saves the new subscription table if the session was already persisted.
func (m *Session) Subscribe(subKey string) {
	m.subscription.Subscribe(subKey)
	m.Subscriptions = *m.subscription.Subscriptions()
	if m.GetKey() != nil {
		m.Save()
	}
}

// Unsubscribe unsubscribes from a database key and saves the new subscription table if the session was already persisted.
func (m *Session) Unsubscribe(subKey string) {
	m.subscription.Unsubscribe(subKey)
	m.Subscriptions = *m.subscription.Subscriptions()
	if m.GetKey() != nil {
		m.Save()
	}
}

// CreateWithID saves Session with the its ID in the default namespace.
func (m *Session) CreateWithID() error {
	m.Expires = time.Now().Add(10 * time.Minute).Unix()
	return m.CreateObject(&database.ApiSessions, m.ID, m)
}

// Create saves Session with the provided name in the default namespace.
func (m *Session) Create(name string) error {
	m.Expires = time.Now().Add(10 * time.Minute).Unix()
	return m.CreateObject(&database.ApiSessions, name, m)
}

// CreateInNamespace saves Session with the provided name in the provided namespace.
func (m *Session) CreateInNamespace(namespace *datastore.Key, name string) error {
	m.Expires = time.Now().Add(10 * time.Minute).Unix()
	return m.CreateObject(namespace, name, m)
}

// Save saves Session.
func (m *Session) Save() error {
	m.Expires = time.Now().Add(10 * time.Minute).Unix()
	return m.SaveObject(m)
}

// GetSession fetches Session with the provided name from the default namespace.
func GetSession(name string) (*Session, error) {
	return GetSessionFromNamespace(&database.ApiSessions, name)
}

// GetSessionFromNamespace fetches Session with the provided name from the provided namespace.
func GetSessionFromNamespace(namespace *datastore.Key, name string) (*Session, error) {
	object, err := database.GetAndEnsureModel(namespace, name, sessionModel)
	if err != nil {
		return nil, err
	}
	model, ok := object.(*Session)
	if !ok {
		return nil, database.NewMismatchError(object, sessionModel)
	}

	if model.subscription == nil {
		model.subscription = database.NewSubscription()
		for _, entry := range model.Subscriptions {
			model.subscription.Subscribe(entry)
		}
	}
	if model.send != nil {
		model.send = make(chan []byte, 1024)
	}

	return model, nil
}

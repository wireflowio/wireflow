package vo

import (
	"linkany/pkg/log"
	"sync"
)

// Message used to wrapper the message for watch
type Message struct {
	EventType EventType
	Nodes     []*NodeVo
	GroupId   uint
	GroupName string
}

func NewMessage(eventType EventType, nodes []*NodeVo, groupId uint, groupName string) *Message {
	return &Message{
		EventType: eventType,
		Nodes:     nodes,
		GroupId:   groupId,
		GroupName: groupName,
	}
}

type EventType int

const (
	EventTypeNodeAdd EventType = iota
	EventTypeNodeRemove
	EventTypeNodeUpdate
	EventTypeGroupAdd
	EventTypeGroupRemove
	EventTypeGroupChanged
)

func (e EventType) String() string {
	switch e {
	case EventTypeNodeAdd:
		return "nodeAdd"
	case EventTypeNodeRemove:
		return "nodeRemove"
	case EventTypeNodeUpdate:
		return "nodeUpdate"
	case EventTypeGroupAdd:
		return "groupAdd"
	case EventTypeGroupRemove:
		return "groupRemove"
	case EventTypeGroupChanged:
		return "groupChanged"
	}
	return "unknown"
}

var lock sync.Mutex
var once sync.Once
var manager *WatchManager

type WatchManager struct {
	lock   sync.Mutex
	m      map[string]chan *Message // key: clientID, value: channel
	logger *log.Logger
}

// NewWatchManager create a whole manager for connected peers
func NewWatchManager() *WatchManager {
	lock.Lock()
	defer lock.Unlock()
	if manager != nil {
		return manager
	}
	once.Do(func() {
		manager = &WatchManager{
			m:      make(map[string]chan *Message),
			logger: log.NewLogger(log.Loglevel, "watchmanager"),
		}
	})

	return manager
}

type RangeFunc func()

func (w *WatchManager) Clientsets() map[string]chan *Message {
	w.lock.Lock()
	defer w.lock.Unlock()

	return w.m
}

// Add adds a new channel to the watch manager for a new connected peer
func (w *WatchManager) Add(key string, ch chan *Message) {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.logger.Verbosef("manager: %v, ch: %v", w, ch)
	w.m[key] = ch
}

// Remove removes a channel from the watch manager for a disconnected peer
func (w *WatchManager) Remove(key string) {
	w.lock.Lock()
	defer w.lock.Unlock()

	delete(w.m, key)
}

// Send sends a message to all connected peer's channel
func (w *WatchManager) Send(key string, msg *Message) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if ch, ok := w.m[key]; ok {
		ch <- msg
	}
}

func (w *WatchManager) Get(key string) chan *Message {
	w.lock.Lock()
	defer w.lock.Unlock()
	ch := w.m[key]
	w.logger.Verbosef("Get channel: %v for node: %v, manager: %v", ch, key, w)
	return ch
}

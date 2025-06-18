package master

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Session struct {
	Channel chan string
}

var Sessions = map[int]*Session{}
var SessionsMutex = sync.RWMutex{}

func AddSession(s *Session) int {
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()
	if s == nil {
		return -1
	}
	id := len(Sessions)
	Sessions[id] = s
	logrus.Debugf("Session %d added", id)
	return id
}

func RemoveSession(id int) {
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()
	if s, ok := Sessions[id]; ok {
		logrus.Debugf("Session %d removed", id)
		close(s.Channel)
		delete(Sessions, id)
	} else {
		logrus.Debugf("Session %d not found", id)
	}
}

func BroadcastMessage(msg string) {
	SessionsMutex.RLock()
	defer SessionsMutex.RUnlock()
	for id, s := range Sessions {
		select {
		case s.Channel <- msg:
			logrus.Debugf("Message sent to session %d", id)
		default:
			logrus.Debugf("Session %d is busy, message not sent", id)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (srv *Service) SendLiveData() {
	SessionsMutex.RLock()
	defer SessionsMutex.RUnlock()
	for id, s := range Sessions {
		select {
		case s.Channel <- srv.AllData.LiveDataToJson():
			logrus.Debugf("Live data sent to session %d", id)
		default:
			logrus.Debugf("Session %d is busy, live data not sent", id)
		}
	}
}

func (srv *Service) wsHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	logrus.Debugf("Websocket request %s %s", r.Method, r.URL.Path)
	if upgrade := r.Header.Get("Upgrade"); upgrade == "websocket" {
		// Handle as websocket request
		err = srv.wsBase(w, r)
	} else {
		// Handle as html page
		err = srv.wsBasePage(w, r)
	}

	if err != nil {
		logrus.WithError(errors.Wrap(err, "Websocket")).Error("Websocket processing error")

		body := fmt.Sprintf(`{"error":[{"message":"%s"}]}`, err.Error())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(body)); err != nil {
			logrus.WithError(errors.Wrap(err, "HTTP")).Error("Response write error")
		}
	}
}

func (mgmt *Service) wsBase(w http.ResponseWriter, r *http.Request) error {
	var err error

	logrus.Debugf("Public connection %s %s", r.Method, r.URL.Path)

	var ws *websocket.Conn
	ws, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.Wrap(err, "Connection upgrade failed")
	}
	logrus.Debug("Client connected")

	ws.SetPongHandler(func(msg string) error {
		logrus.Tracef("Ping/pong received %s", msg)
		return nil
	})

	session := Session{
		Channel: make(chan string, 10), // Buffered channel to handle messages
	}
	id := AddSession(&session)
	if id < 0 {
		return errors.New("Failed to add session")
	}

	// Websocket message sender thread
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Websocket write panic")
			}
		}()
		for msg := range session.Channel {
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				logrus.WithError(errors.Wrap(err, "Websocket write")).Error("Failed to send message")
				return
			}
			logrus.Debugf("Sent message to session %d: %s", id, string(msg))
		}
	}()

	for {
		var reqType int
		var req []byte
		reqType, req, err = ws.ReadMessage()
		if err != nil {
			// 1000 normal closure
			// 1001 going away
			// 1006 abnormal closure
			if websocket.IsCloseError(err, 1000, 1001, 1006) {
				logrus.Debugf("Websocket closed\n%s", err.Error())
				err = nil
			} else {
				err = errors.Wrap(err, "ReadMessages")
			}

			break
		}

		if reqType == websocket.TextMessage {
			BroadcastMessage(string(req)) // Broadcast to all sessions

		}
	}
	RemoveSession(id) // Clean up session on exit

	return err
}

func (s *Service) wsBasePage(w http.ResponseWriter, r *http.Request) error {
	logrus.Debug("Websocket page requested")
	body := `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
		<meta http-equiv="X-UA-Compatible" content="ie=edge" />
		<title>WebSocket Playground</title>
	</head>
	<body>
		<h2>Websocket Tester</h2>
		<form name="publish">
			<input type="text" name="message">
			<input type="submit" value="Send">
		</form>

		<div id="messages"></div>
		<script>
			let socket = new WebSocket(((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/ws");
			console.log("Attempting Connection...");

			document.forms.publish.onsubmit = function() {
				let msg = this.message.value;
				socket.send(msg);
				return false;
			};

			socket.onmessage = function(event) {
				let msg = event.data;
				let messageElem = document.createElement('div');
				messageElem.textContent = msg;
				document.getElementById('messages').prepend(messageElem);
			}

			socket.onopen = () => {
				console.log("Successfully Connected");
			};

			socket.onclose = event => {
				console.log("Socket Closed Connection: ", event);
			};

			socket.onerror = error => {
				console.log("Socket Error: ", error);
			};

		</script>
	</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(body)); err != nil {
		return errors.Wrap(err, "HTTP")
	}
	return nil
}

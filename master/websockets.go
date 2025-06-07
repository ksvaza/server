package master

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (srv *Service) wsHandler(w http.ResponseWriter, r *http.Request) {
	var err error
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

	logrus.Debug(`Public connection`)

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

		resType := websocket.TextMessage
		var res string

		if reqType == websocket.TextMessage {
			res = "Public echo " + string(req)

		} else {
			err = errors.New("Only text message supported")
		}

		logrus.Debugf("Responded %s", string(res))
		if err = ws.WriteMessage(resType, []byte(res)); err != nil {
			err = errors.Wrap(err, "WS")
			break
		}
	}
	return err
}

func (s *Service) wsBasePage(w http.ResponseWriter, r *http.Request) error {
	body := `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
		<meta http-equiv="X-UA-Compatible" content="ie=edge" />
		<title>WebSocket Playground</title>
	</head>
	<body>
		<h2>Hello Aranet</h2>
		<form name="publish">
			<input type="text" name="message">
			<input type="submit" value="Send">
		</form>

		<div id="messages"></div>
		<script>
			let socket = new WebSocket(((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/base");
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
				socket.send("Hello!")
			};

			socket.onclose = event => {
				console.log("Socket Closed Connection: ", event);
				socket.send("Client Closed!")
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

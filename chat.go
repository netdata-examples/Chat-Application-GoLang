package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
	"github.com/googollee/go-socket.io"
)

// Should look like path
const websocketRoom = "/chat"

func main() {
	lastMessages := []string{}
	var lmMutex sync.Mutex
	// Sets the number of maxium goroutines to the 2*numberCPU + 1
	runtime.GOMAXPROCS((runtime.NumCPU() * 2) + 1)

	// Configuring socket.io Server
	sio, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	sio.On("connection", func(so socketio.Socket) {
		var user string
		log.Printf("%s on connected", so.Id())

		so.Join(websocketRoom)

		lmMutex.Lock()
		for i, _ := range lastMessages {
			so.Emit("message", lastMessages[i])
		}
		lmMutex.Unlock()

		so.On("joined_user", func(username string) {
			user = username
			log.Printf("%s has username is %s", so.Id(), username)
		})

		so.On("send_message", func(message string) {
			log.Printf("Sent message from %s: %s", user, message)

			res := map[string]interface{} {
				"username": user,
				"message":  message,
				"dateTime": time.Now().UnixNano() / int64(time.Millisecond),
				"type":     "message",
			}

			jsonRes, _ := json.Marshal(res)

			lmMutex.Lock()
			if len(lastMessages) == 100 {
				lastMessages = lastMessages[1:100]
			}
			lastMessages = append(lastMessages, string(jsonRes))
			lmMutex.Unlock()

			so.Emit("message", string(jsonRes))
			so.BroadcastTo(websocketRoom, "message", string(jsonRes))

			soapRequestContent := "<?xml version=\"1.0\" encoding=\"utf-8\"?><soap:Envelope xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:soap=\"http://schemas.xmlsoap.org/soap/envelope/\"><soap:Body><InsertRecord xmlns=\"http://tempuri.org/\"><APIKey>1a31b926</APIKey><InsertList><AccPoKeyValuePair><Key>dc_UserName</Key><Value>"+user+"</Value></AccPoKeyValuePair><AccPoKeyValuePair><Key>dc_Message</Key><Value>"+message+"</Value></AccPoKeyValuePair><AccPoKeyValuePair><Key>dc_MessageTo</Key><Value>all</Value></AccPoKeyValuePair></InsertList></InsertRecord></soap:Body></soap:Envelope>"

			httpClient := new(http.Client)
    	_, err := httpClient.Post("http://www.netdata.com/AccPo.asmx", "text/xml; charset=utf-8", bytes.NewBufferString(soapRequestContent))

			if err != nil {
				log.Fatal(err)
			}
		})

		so.On("disconnection", func() {
			log.Printf("%s on disconnect", so.Id())
		})
	})

	sio.On("error", func(so socketio.Socket, err error) {
		log.Println("Error: ", err)
	})

	// Sets up the handlers and listen on port 8080
	http.Handle("/socket.io/", sio)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.Handle("/", http.FileServer(http.Dir("./templates/")))

	// Default to :8080 if not defined via environmental variable.
	listen := os.Getenv("LISTEN")

	if listen == "" {
		listen = ":3001"
	}

	log.Printf("Application is running on %s", listen)
	http.ListenAndServe(listen, nil)
}

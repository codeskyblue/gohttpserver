# Package information

The iris-conrib/websocket package has been converted from the [gorilla/websocket](https://github.com/gorilla/websocket) (in order to make it work with Iris).


If you need more, use the high [high level iris-ws](https://github.com/kataras/iris/tree/master/websocket), I recommend you use that instead.


# Websockets

**WebSocket is a protocol providing full-duplex communication channels over a single TCP connection**. The WebSocket protocol was standardized by the IETF as RFC 6455 in 2011, and the WebSocket API in Web IDL is being standardized by the W3C.

WebSocket is designed to be implemented in web browsers and web servers, but it can be used by any client or server application. The WebSocket Protocol is an independent TCP-based protocol. Its only relationship to HTTP is that its handshake is interpreted by HTTP servers as an Upgrade request. The WebSocket protocol makes more interaction between a browser and a website possible, **facilitating the real-time data transfer from and to the server**.

[Read more about Websockets](https://en.wikipedia.org/wiki/WebSocket)

-----

How to use

```go
import (
	"github.com/iris-contrib/websocket"
	"github.com/kataras/iris"
)

func chat(c *websocket.Conn) {
	// defer c.Close()
	// mt, message, err := c.ReadMessage()
	// c.WriteMessage(mt, message)
}

var upgrader = websocket.New(chat) // use default options
//var upgrader = websocket.Custom(chat, 1024, 1024) // customized options, read and write buffer sizes (int). Default: 4096
// var upgrader = websocket.New(chat).DontCheckOrigin() // it's useful when you have the websocket server on a different machine

func myChatHandler(ctx *iris.Context) {
	err := upgrader.Upgrade(ctx)// returns only error, executes the handler you defined on the websocket.New before (the 'chat' function)
}

func main() {
  iris.Get("/chat_back", myChatHandler)
  iris.Listen(":80")
}

```

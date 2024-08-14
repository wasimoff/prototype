package transport

import (
	"fmt"
	"net/http"
	"wasimoff/broker/net/pb"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Subprotocol between a Provider and the Broker defines the concrete codec.
var (
	provider_v1_protobuf = pb.Subprotocol_wasimoff_provider_v1_protobuf.String()
	provider_v1_json     = pb.Subprotocol_wasimoff_provider_v1_json.String()
)

// WebSocketTransport implements broker/net/transport.Transport for Messaging
type WebSocketTransport struct {
	conn *websocket.Conn // upgraded WebSocket connection
	req  *http.Request   // original http.Request
}

// UpgradeToWebSocketTransport can be used inside a http.HanderFunc to upgrade the
// connection to a WebSocket and instantiate a Transport for Messaging
func UpgradeToWebSocketTransport(w http.ResponseWriter, r *http.Request, origins []string) (t *WebSocketTransport, err error) {
	defer wraperr(&err, "upgrade failed: %w")

	// upgrade the connection specifying our wasimoff/provider/v1 subprotocols
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{
			// in order of preference, though we shouldn't rely on that
			provider_v1_protobuf,
			provider_v1_json,
		},
		// TODO: check if slices.Contains(origins, "*") and prevent in production
		OriginPatterns: origins,
	})
	if err != nil {
		return
	}
	if conn.Subprotocol() == "" {
		// reject unsupported (empty) subprotocol
		conn.Close(websocket.StatusProtocolError, "must use a supported wasimoff.provider.v1+codec subprotocol")
		return nil, fmt.Errorf("unsupported subprotocol")
	}

	// return the Transport
	return &WebSocketTransport{conn, r}, nil
}

// WriteMessage will marshal the given message using the negotiated subprotocol codec
// and send it over the WebSocket connection. It is safe for concurrent writes.
func (ws *WebSocketTransport) WriteMessage(message *pb.Envelope) (err error) {
	defer wraperr(&err, "transport write: %w")

	var b []byte
	var mt websocket.MessageType
	// marshal using the correct codec for protocol
	switch ws.conn.Subprotocol() {

	case provider_v1_protobuf:
		mt = websocket.MessageBinary
		b, err = proto.Marshal(message)

	case provider_v1_json:
		mt = websocket.MessageText
		b, err = protojson.Marshal(message)

	default:
		// shouldn't happen if connection upgrade works correctly
		ws.conn.Close(websocket.StatusProtocolError, "broker ended up with an unsupported protocol")
		return fmt.Errorf("unknown subprotocol in transport: %s", ws.conn.Subprotocol())
	}
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	// write bytes to socket
	err = ws.conn.Write(ws.req.Context(), mt, b)
	if err != nil {
		err = fmt.Errorf("websocket: %w", err)
	}
	return
}

// WriteMessage will read the next message from the WebSocket connection and unmarshal
// the message using the negotiated subprotocol codec. NOT safe for concurrent reads,
// so you need to synchronize yourself or limit to a single reader.
func (ws *WebSocketTransport) ReadMessage(message *pb.Envelope) (err error) {
	defer wraperr(&err, "transport read: %w")

	// read bytes from socket
	mt, b, err := ws.conn.Read(ws.req.Context())
	if err != nil {
		return fmt.Errorf("websocket: %w", err)
	}

	expectFormat := func(expected websocket.MessageType) error {
		if mt != expected {
			e := fmt.Sprintf("sent %s to a %s transport", mt, ws.conn.Subprotocol())
			ws.conn.Close(websocket.StatusUnsupportedData, e)
			return fmt.Errorf("wrong message type: %s", e)
		}
		return nil
	}

	// try to unmarshal the message depending on subprotocol
	switch {

	case ws.conn.Subprotocol() == provider_v1_protobuf:
		if err = expectFormat(websocket.MessageBinary); err != nil {
			return
		}
		err = proto.Unmarshal(b, message)

	case ws.conn.Subprotocol() == provider_v1_json:
		if err = expectFormat(websocket.MessageText); err != nil {
			return
		}
		err = protojson.Unmarshal(b, message)

	default:
		// shouldn't happen if connection upgrade works correctly
		ws.conn.Close(websocket.StatusProtocolError, "broker ended up with an unsupported protocol")
		return fmt.Errorf("unknown subprotocol in transport: %s", ws.conn.Subprotocol())
	}
	if err != nil {
		err = fmt.Errorf("unmarshal: %s", err)
	}
	return
}

// Close the WebSocket connection with an orderly handshake
func (ws *WebSocketTransport) Close() error {
	return ws.conn.Close(websocket.StatusNormalClosure, "broker is closing this transport, bye.")
}

// wraperr wraps an error, if it is not nil
func wraperr(err *error, format string) {
	if *err != nil {
		*err = fmt.Errorf(format, *err)
	}
}

package transport

import (
	"context"
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
func UpgradeToWebSocketTransport(w http.ResponseWriter, req *http.Request, origins []string) (t *WebSocketTransport, err error) {
	defer wraperr(&err, "upgrade failed: %w")

	// subprotocols in order of preference, upgrade will pick first
	protocols := []string{
		provider_v1_protobuf,
		provider_v1_json,
	}

	// upgrade the connection to create a socket
	conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
		Subprotocols:   protocols,
		OriginPatterns: origins,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnection, err)
	}
	if conn.Subprotocol() == "" {
		// reject unsupported (empty) subprotocol
		conn.Close(websocket.StatusProtocolError, fmt.Sprintf("supported protocols: %v", protocols))
		return nil, fmt.Errorf("%w: no supported subprotocol", ErrCodec)
	}

	// return the Transport
	return &WebSocketTransport{conn, req}, nil
}

// -------------------- read / write -------------------- >>

// WriteMessage will marshal the given message using the negotiated subprotocol codec
// and send it over the WebSocket connection. It is safe for concurrent writes.
func (ws *WebSocketTransport) WriteMessage(ctx context.Context, message *pb.Envelope) (err error) {
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
		// shouldn't happen if connection upgrade worked correctly
		ws.conn.Close(websocket.StatusProtocolError, "broker ended up with an unknown protocol")
		return fmt.Errorf("%w: unknown subprotocol in transport: %s", ErrCodec, ws.conn.Subprotocol())
	}
	if err != nil {
		return fmt.Errorf("%w: marshal: %w", ErrCodec, err)
	}
	// write bytes to socket
	err = ws.conn.Write(ctx, mt, b)
	if err != nil {
		err = fmt.Errorf("%s: websocket: %w", ErrConnection, err)
	}
	return
}

// ReadMessage will read the next message from the WebSocket connection and unmarshal
// the message using the negotiated subprotocol codec. NOT safe for concurrent reads,
// so you need to synchronize yourself or limit to a single reader.
func (ws *WebSocketTransport) ReadMessage(ctx context.Context, message *pb.Envelope) (err error) {
	defer wraperr(&err, "transport read: %w")

	// read bytes from socket
	mt, b, err := ws.conn.Read(ctx)
	if err != nil {
		return fmt.Errorf("%w: websocket: %w", ErrConnection, err)
	}

	// lambda to expect a certain message type depending on the protocol
	expectFormat := func(expected websocket.MessageType) error {
		if mt != expected {
			cause := fmt.Sprintf("sent %s to a %s transport", mt, ws.conn.Subprotocol())
			err := fmt.Errorf("%w: wrong message type: %s", ErrCodec, cause)
			ws.conn.Close(websocket.StatusUnsupportedData, cause)
			return err
		}
		return nil
	}

	// try to unmarshal the message depending on subprotocol
	switch {

	case ws.conn.Subprotocol() == provider_v1_protobuf:
		if err = expectFormat(websocket.MessageBinary); err != nil {
			return err
		}
		err = proto.Unmarshal(b, message)

	case ws.conn.Subprotocol() == provider_v1_json:
		if err = expectFormat(websocket.MessageText); err != nil {
			return err
		}
		err = protojson.Unmarshal(b, message)

	default:
		// shouldn't happen if connection upgrade worked correctly
		ws.conn.Close(websocket.StatusProtocolError, "broker ended up with an unknown protocol")
		return fmt.Errorf("%w: unknown subprotocol in transport: %s", ErrCodec, ws.conn.Subprotocol())
	}
	if err != nil {
		err = fmt.Errorf("%w: unmarshal: %s", ErrCodec, err)
	}
	return
}

// -------------------- misc -------------------- >>

// Return the remote Addr from initial http.Request
func (ws *WebSocketTransport) Addr() string {
	return ProxiedAddr(ws.req)
}

// Close the WebSocket connection with an orderly handshake.
func (ws *WebSocketTransport) Close(cause error) {
	if cause == nil {
		ws.conn.Close(websocket.StatusNormalClosure, "bye!")
	} else {
		err := cause.Error()
		ws.conn.Close(websocket.StatusInternalError, err[:min(len(err), 125)])
	}
}

// wraperr wraps an error, if it is not nil
func wraperr(err *error, format string) {
	if *err != nil {
		*err = fmt.Errorf(format, *err)
	}
}

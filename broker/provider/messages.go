package provider

import (
	"io"
	"log"

	"github.com/mitchellh/mapstructure"
	"github.com/vmihailenco/msgpack/v5"
)

// --------------- Message channel for control ---------------

// Message is the MessagePack encoding of control messages on the wire
type Message struct {
	Type    string `msgpack:"type"`
	Message any    `msgpack:"message"`
}

// MessageChannel is a chan of decoded messages and a utility function to send
type MessageChannel struct {
	stream   io.ReadWriteCloser
	encoder  *msgpack.Encoder
	closing  bool
	Messages chan Message
}

func NewMessageChannel(stream io.ReadWriteCloser) *MessageChannel {
	m := &MessageChannel{
		stream:   stream,
		encoder:  msgpack.NewEncoder(stream),
		Messages: make(chan Message),
	}
	go m.input()
	return m
}

// input decodes incoming messages and puts them into the channel
func (m *MessageChannel) input() {
	decoder := msgpack.NewDecoder(m.stream)
	var err error
	for err == nil {
		var message Message
		if err = decoder.Decode(&message); err == nil {
			m.Messages <- message
		}
	}
	if !m.closing { // stop from logging when purposefully closing
		log.Printf("failed to decode message: %s", err)
	}
	m.Close()
	close(m.Messages)
}

// Send .. sends a Message to the Provider over the channel
func (m *MessageChannel) Send(t string, message any) error {
	if m.closing {
		return io.ErrClosedPipe
	}
	return m.encoder.Encode(Message{t, message})
}

func (m *MessageChannel) Close() error {
	if m.closing {
		return nil
	}
	m.closing = true
	return m.stream.Close()
}

// --------------- handle incoming messages ---------------

// RunMessageHandler should be started in a goroutine to handle incoming messages
func (p *Provider) RunMessageHandler() {
	for message := range p.msgchan.Messages {
		// switch on message type and decode message to appropriate struct
		switch message.Type {

		// hello world
		case helloWorldMessageKey:
			var body helloWorldMessage
			if err := mapstructure.Decode(message.Message, &body); err != nil {
				log.Printf("[%s] failed to decode as HelloWorldMessage: %#v", p.Addr, message)
				continue
			}
			log.Printf("[%s] HelloWorld sent on %s", p.Addr, body.Sent)

		// updated provider information
		case providerInfoMessageKey:
			var body providerInfoMessage
			if err := mapstructure.Decode(message.Message, &body); err != nil {
				log.Printf("[%s] failed to decode as ProviderInfoMessage: %#v", p.Addr, message)
				continue
			}
			p.Info.mutex.Lock()
			p.Info.Name = body.Name
			p.Info.Platform = body.Platform
			p.Info.UserAgent = body.UserAgent
			p.Info.mutex.Unlock()
			log.Printf("[%s] Updated Info: { platform: %q, useragent: %q }", p.Addr, body.Platform, body.UserAgent)

		// updated worker pool information
		case providerPoolInfoMessageKey:
			var body providerPoolInfoMessage
			if err := mapstructure.Decode(message.Message, &body); err != nil {
				log.Printf("[%s] failed to decode as ProviderPoolInfoMessage: %#v", p.Addr, message)
				continue
			}
			p.Info.mutex.Lock()
			p.Info.Max = body.Max
			p.Info.Pool = body.Pool
			// TODO: set concurrency limit higher than number of workers to queue before the previous returns?
			p.limiter.SetLimit(max(0, body.Pool))
			p.Info.mutex.Unlock()
			log.Printf("[%s] Workers: { pool: %d / %d }", p.Addr, body.Pool, body.Max)

		default:
			log.Printf("[%s] Received unknown message type: %#v", p.Addr, message)

		}
	}
	// message channel died, close the stream
	p.Close()
}

// --------------- types of expected messages ---------------

const helloWorldMessageKey = "hello"

type helloWorldMessage struct {
	Sent string `mapstructure:"sent"`
}

const providerInfoMessageKey = "providerinfo"

type providerInfoMessage struct {
	// Name is a logging-friendly name for the Provider
	Name string `mapstructure:"name"`
	// Platform is coming from `navigator.platform` in the browser
	Platform string `mapstructure:"platform"`
	// UserAgent is coming from `navigator.useragent` in the browser
	UserAgent string `mapstructure:"useragent"`
}

const providerPoolInfoMessageKey = "poolinfo"

type providerPoolInfoMessage struct {
	// Max is the maximum possible hardware concurrency reported by this Provider
	Max int `mapstructure:"nmax"`
	// Pool is the *current* pool capacity, i.e. number of threads or runners
	Pool int `mapstructure:"pool"`
}

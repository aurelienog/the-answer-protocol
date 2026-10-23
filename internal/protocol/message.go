package protocol

import (
	"encoding/json"
	"fmt"
)

type Kind uint8

const (
	KindOK Kind = iota
	KindERR
	KindEVT
)

// Message representa un mensaje TAP de salida.
type Message struct {
	Kind      Kind
	Payload   string
	Code      int
	Name      string
	EventType string
	Data      string
}

// OK crea una respuesta correcta con texto plano.
// Ejemplo: OK connected
func OK(payload string) Message {
	return Message{
		Kind:    KindOK,
		Payload: payload,
	}
}

// OKJSON crea una respuesta correcta cuyo contenido es JSON.
// Ejemplo: OK {"id":"42_cluster","name":"Cluster"}
func OKJSON(value any) (Message, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return Message{}, fmt.Errorf("encode TAP JSON response: %w", err)
	}

	return Message{
		Kind:    KindOK,
		Payload: string(data),
	}, nil
}

// ERR crea una respuesta de error.
// Ejemplo: ERR 201 NAME_IN_USE
func ERR(code int, name string) Message {
	return Message{
		Kind: KindERR,
		Code: code,
		Name: name,
	}
}

// EVT crea un evento asíncrono.
// Ejemplo: EVT GLOBAL_CHAT alice: Hello!
func EVT(eventType, data string) Message {
	return Message{
		Kind:      KindEVT,
		EventType: eventType,
		Data:      data,
	}
}
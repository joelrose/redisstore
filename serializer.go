package redisstore

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/gorilla/sessions"
)

// SessionSerializer provides an interface for alternative serializers.
type SessionSerializer interface {
	Deserialize(d []byte, ss *sessions.Session) error
	Serialize(ss *sessions.Session) ([]byte, error)
}

// JSONSerializer encodes the session map to JSON.
type JSONSerializer struct{}

var _ SessionSerializer = (*JSONSerializer)(nil)

// Serialize to JSON. All keys must be strings.
func (s JSONSerializer) Serialize(ss *sessions.Session) ([]byte, error) {
	m := make(map[string]interface{}, len(ss.Values))
	for k, v := range ss.Values {
		ks, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("json: non-string key value, cannot serialize session values: %v", k)
		}
		m[ks] = v
	}

	contents, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("json: serializing session values: %v", err)
	}

	return contents, nil
}

// Deserialize back to map[string]interface{}.
func (s JSONSerializer) Deserialize(d []byte, ss *sessions.Session) error {
	m := make(map[string]interface{})
	if err := json.Unmarshal(d, &m); err != nil {
		return fmt.Errorf("json: deserializing session values: %v", err)
	}

	if ss.Values == nil {
		ss.Values = make(map[interface{}]interface{}, len(m))
	}

	for k, v := range m {
		ss.Values[k] = v
	}

	return nil
}

// GobSerializer uses the gob package to encode the session map.
type GobSerializer struct{}

var _ SessionSerializer = (*GobSerializer)(nil)

// Serialize using gob.
func (s GobSerializer) Serialize(ss *sessions.Session) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)

	if err := enc.Encode(ss.Values); err != nil {
		return nil, fmt.Errorf("gob: encoding session values: %v", err)
	}

	return buf.Bytes(), nil
}

// Deserialize back to map[interface{}]interface{}.
func (s GobSerializer) Deserialize(d []byte, ss *sessions.Session) error {
	dec := gob.NewDecoder(bytes.NewBuffer(d))

	if err := dec.Decode(&ss.Values); err != nil {
		return fmt.Errorf("gob: decoding session values: %v", err)
	}

	return nil
}

package redisstore

import (
	"encoding/gob"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
)

type object struct {
	Name string
}

func TestJSONSerializer(t *testing.T) {
	serializer := &JSONSerializer{}

	t.Run("serialize and deserialize", func(t *testing.T) {
		give := &sessions.Session{
			ID:    "ID",
			IsNew: true,
			Values: map[interface{}]interface{}{
				"string": "value",
				"number": 54321,
				"object": object{Name: "object"},
				"array":  []string{"a", "b", "c"},
				"map":    map[string]string{"a": "b", "c": "d"},
				"nil":    nil,
			},
			Options: &sessions.Options{},
		}

		want := &sessions.Session{
			Values: map[interface{}]interface{}{
				"string": "value",
				"number": float64(54321),
				"object": map[string]interface{}{"Name": "object"},
				"array":  []interface{}{"a", "b", "c"},
				"map":    map[string]interface{}{"a": "b", "c": "d"},
				"nil":    nil,
			},
		}

		serialized, err := serializer.Serialize(give)

		assert.NoError(t, err)
		assert.NotEmpty(t, serialized)

		session := &sessions.Session{}
		err = serializer.Deserialize(serialized, session)

		assert.NoError(t, err)
		assert.Equal(t, want, session)
	})

	t.Run("serialize with non string key", func(t *testing.T) {
		input := &sessions.Session{
			Values: map[interface{}]interface{}{
				12345: "value",
			},
		}

		serialized, err := serializer.Serialize(input)

		assert.Error(t, err)
		assert.Empty(t, serialized)
	})
}

func TestGobSerializer(t *testing.T) {
	serializer := &GobSerializer{}

	t.Run("serialize and deserialize", func(t *testing.T) {
		gob.Register(object{})
		gob.Register(map[string]string{})

		give := &sessions.Session{
			ID:    "ID",
			IsNew: true,
			Values: map[interface{}]interface{}{
				"string": "value",
				"number": 54321,
				"object": object{Name: "object"},
				"array":  []string{"a", "b", "c"},
				"map":    map[string]string{"a": "b", "c": "d"},
				"nil":    nil,
			},
			Options: &sessions.Options{},
		}

		want := &sessions.Session{
			Values: map[interface{}]interface{}{
				"string": "value",
				"number": 54321,
				"object": object{Name: "object"},
				"array":  []string{"a", "b", "c"},
				"map":    map[string]string{"a": "b", "c": "d"},
				"nil":    nil,
			},
		}

		serialized, err := serializer.Serialize(give)

		assert.NoError(t, err)
		assert.NotEmpty(t, serialized)

		session := &sessions.Session{}
		err = serializer.Deserialize(serialized, session)

		assert.NoError(t, err)
		assert.Equal(t, want, session)
	})
}

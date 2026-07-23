package grpc_server

import (
	"testing"

	"github.com/moveaxlab/go-grpc-server/internal"
	"github.com/stretchr/testify/assert"
)

func TestRedact(t *testing.T) {
	t.Run("non-sensitive fields are left untouched", func(t *testing.T) {
		out := Redact(&internal.Input{Value: "Hello"})

		assert.Equal(t, map[string]interface{}{"value": "Hello"}, out)
	})

	t.Run("non-proto requests are returned unchanged", func(t *testing.T) {
		out := Redact("plain string")

		assert.Equal(t, "plain string", out)
	})

	t.Run("fields marked debug_redact are redacted", func(t *testing.T) {
		out := Redact(&internal.SensitiveInput{
			Username: "alice",
			Password: "hunter2",
		})

		assert.Equal(t, map[string]interface{}{
			"username": "alice",
			"password": redactedPlaceholder,
		}, out)
	})

	t.Run("debug_redact works for non-string fields", func(t *testing.T) {
		out := Redact(&internal.SensitiveInput{
			Username: "alice",
			Pin:      1234,
		})

		assert.Equal(t, map[string]interface{}{
			"username": "alice",
			"pin":      redactedPlaceholder,
		}, out)
	})

	t.Run("a message field marked debug_redact collapses to a single placeholder", func(t *testing.T) {
		out := Redact(&internal.SensitiveInput{
			Username:     "alice",
			SecretNested: &internal.Nested{Public: "p", Secret: "s"},
		})

		assert.Equal(t, map[string]interface{}{
			"username":      "alice",
			"secret_nested": redactedPlaceholder,
		}, out)
	})

	t.Run("debug_redact is honored inside nested messages", func(t *testing.T) {
		out := Redact(&internal.SensitiveInput{
			Nested: &internal.Nested{Public: "ok", Secret: "s3cr3t"},
		})

		assert.Equal(t, map[string]interface{}{
			"nested": map[string]interface{}{
				"public": "ok",
				"secret": redactedPlaceholder,
			},
		}, out)
	})

	t.Run("repeated and map fields marked debug_redact collapse to a single placeholder", func(t *testing.T) {
		out := Redact(&internal.SensitiveInput{
			Username: "alice",
			Tokens:   []string{"a", "b"},
			Secrets:  map[string]string{"first": "x", "second": "y"},
		})

		assert.Equal(t, map[string]interface{}{
			"username": "alice",
			"tokens":   redactedPlaceholder,
			"secrets":  redactedPlaceholder,
		}, out)
	})

	t.Run("debug_redact is honored inside repeated and map fields", func(t *testing.T) {
		out := Redact(&internal.SensitiveInput{
			Items: []*internal.Nested{
				{Public: "a", Secret: "x"},
				{Public: "b", Secret: "y"},
			},
			ByKey: map[string]*internal.Nested{
				"first": {Public: "c", Secret: "z"},
			},
		})

		assert.Equal(t, map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"public": "a", "secret": redactedPlaceholder},
				map[string]interface{}{"public": "b", "secret": redactedPlaceholder},
			},
			"by_key": map[string]interface{}{
				"first": map[string]interface{}{"public": "c", "secret": redactedPlaceholder},
			},
		}, out)
	})
}

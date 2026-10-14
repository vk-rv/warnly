package warnly_test

import (
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestEventBodyJSONRoundTrip(t *testing.T) {
	t.Parallel()
	for _, timestamp := range []string{`1234567890.5`, `"2009-02-13T23:31:30.5Z"`} {
		for _, wrapped := range []bool{false, true} {
			exceptions := `[{"type":"Error","value":"failure","stacktrace":{"frames":[]}}]`
			threads := `[{"id":"1","current":true,"stacktrace":{"frames":[]}}]`
			if wrapped {
				exceptions = `{"values":` + exceptions + `}`
				threads = `{"values":` + threads + `}`
			}
			payload := `{"timestamp":` + timestamp + `,"exception":` + exceptions + `,"threads":` + threads + `}`
			var event warnly.EventBody
			require.NoError(t, json.Unmarshal([]byte(payload), &event))
			require.True(t, event.Timestamp.Equal(time.Unix(1234567890, 500000000)))
			require.Len(t, event.Exception, 1)
			require.Equal(t, "failure", event.Exception[0].Value)
			require.Len(t, event.Threads, 1)
			require.True(t, event.Threads[0].Current)
			encoded, err := json.Marshal(event)
			require.NoError(t, err)
			var restored warnly.EventBody
			require.NoError(t, json.Unmarshal(encoded, &restored))
			require.True(t, event.Timestamp.Equal(restored.Timestamp.Time))
			require.Equal(t, event.Exception, restored.Exception)
			require.Equal(t, event.Threads, restored.Threads)
		}
	}
}

func TestUint8ArrayJSONRoundTrip(t *testing.T) {
	t.Parallel()
	original := warnly.Uint8Array{0, 1, 255}
	encoded, err := json.Marshal(original)
	require.NoError(t, err)
	require.Equal(t, `[0,1,255]`, string(encoded))
	var restored warnly.Uint8Array
	require.NoError(t, json.Unmarshal(encoded, &restored))
	require.Equal(t, original, restored)
}

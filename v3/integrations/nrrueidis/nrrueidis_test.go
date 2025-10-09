package nrrueidis

import (
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
)

func Test_nrrueidisHook(t *testing.T) {
	tests := []struct {
		name  string
		opt   rueidis.ClientOption
		check func(t *testing.T, h *hook) error
	}{
		{
			name: "without options",
			opt:  rueidis.ClientOption{},
			check: func(t *testing.T, h *hook) error {
				assert.NotNil(t, h)

				// Hook should have the datastore segment, but no details on the actual target.
				assert.Equal(t, newrelic.DatastoreRedis, h.segment.Product)
				assert.Empty(t, h.segment.Host)

				return nil
			},
		},
		{
			name: "with init address",
			opt: rueidis.ClientOption{
				InitAddress: []string{"redis:6379"},
			},
			check: func(t *testing.T, h *hook) error {
				assert.NotNil(t, h)

				// Hook should have the datastore segment and details on the actual target.
				assert.Equal(t, newrelic.DatastoreRedis, h.segment.Product)
				assert.Equal(t, "redis", h.segment.Host)
				assert.Equal(t, "6379", h.segment.PortPathOrID)

				return nil
			},
		},
		{
			name: "with port-only init address",
			opt: rueidis.ClientOption{
				InitAddress: []string{":6379"},
			},
			check: func(t *testing.T, h *hook) error {
				assert.NotNil(t, h)

				// Hook should have the datastore segment and details on the actual target.
				assert.Equal(t, newrelic.DatastoreRedis, h.segment.Product)
				assert.Equal(t, "localhost", h.segment.Host)
				assert.Equal(t, "6379", h.segment.PortPathOrID)

				return nil
			},
		},
		{
			name: "with localhost init address",
			opt: rueidis.ClientOption{
				InitAddress: []string{"localhost:6379"},
			},
			check: func(t *testing.T, h *hook) error {
				assert.NotNil(t, h)

				// Hook should have the datastore segment and details on the actual target.
				assert.Equal(t, newrelic.DatastoreRedis, h.segment.Product)
				assert.Equal(t, "localhost", h.segment.Host)
				assert.Equal(t, "6379", h.segment.PortPathOrID)

				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := nrrueidisHook(tt.opt)
			err := tt.check(t, hook)
			assert.NoError(t, err)
		})
	}
}

package fields_test

import (
	"testing"

	"github.com/belastingdienst/opr-paas/internal/fields"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	entries1 = fields.Entries{
		"paas1": fields.Elements{
			"key1": "value1",
			"key2": 2,
		},
	}
	entries2 = fields.Entries{
		"paas1": fields.Elements{
			"key1": "othervalue1",
			"key4": "othervalue4",
		},
		"paas2": fields.Elements{
			"key1": "value1",
			"key3": 3,
		},
	}
)

// internal/fields/entries.go						26-31 37-39 49-51 55-58
func TestHashData(t *testing.T) {
	resultEntries := entries1.Merge(entries2)
	assert.Len(t, resultEntries, 2)
	require.Contains(t, resultEntries, "paas1")
	elements1 := resultEntries["paas1"]
	assert.Len(t, elements1, 3)
	require.Contains(t, elements1, "key1")
	assert.Equal(t, elements1["key1"], "othervalue1")
	require.Contains(t, elements1, "key2")
	assert.Equal(t, elements1["key2"], 2)
	assert.Contains(t, elements1, "key4")
	require.Contains(t, resultEntries, "paas2")
	elements2 := resultEntries["paas2"]
	assert.Len(t, elements2, 2)
}

func TestAsString(t *testing.T) {
	// revive:disable-next-line
	assert.Equal(t, "{ 'paas1': { 'key1': 'othervalue1', 'key2': '2', 'key4': 'othervalue4' }, 'paas2': { 'key1': 'value1', 'key3': '3' } }", entries1.String())
	// revive:disable-next-line
	assert.Equal(t, "{ 'paas1': { 'key1': 'othervalue1', 'key4': 'othervalue4' }, 'paas2': { 'key1': 'value1', 'key3': '3' } }", entries2.String())
}

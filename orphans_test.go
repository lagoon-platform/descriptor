package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrphans(t *testing.T) {
	refs := CreateOrphans()
	refs.new("ref1", "p")
	refs.new("ref2", "p")
	refs.new("ref2", "p")
	refs.new("ref2", "s")
	// Duplicated ref should not be added
	assert.Len(t, refs.Refs, 3)
	assert.Contains(t, refs.Refs, "ref1-p", "ref2-p", "ref2-s")

	others := CreateOrphans()
	others.new("ref3", "p")
	others.new("ref4", "p")
	others.new("ref5", "p")
	// Duplicated ref already defined into refs should not be added
	others.new("ref2", "p")

	for id := range others.Refs {
		refs.AddReference(id)
	}

	assert.Len(t, refs.Refs, 6)
	assert.Contains(t, refs.Refs, "ref1-p", "ref2-p", "ref3-p", "ref4-p", "ref5-p", "ref2-s")

	refs.NoMoreAnOrhpan("ref1-p")

	assert.Len(t, refs.Refs, 5)
	assert.Contains(t, refs.Refs, "ref2-p", "ref3-p", "ref4-p", "ref5-p", "ref2-s")

}

func TestOrphansKeyType(t *testing.T) {
	refs := CreateOrphans()
	k, kind := refs.KeyType("key-type")
	assert.Equal(t, "key", k)
	assert.Equal(t, "type", kind)

	k, kind = refs.KeyType("key-subKey-type")
	assert.Equal(t, "key-subKey", k)
	assert.Equal(t, "type", kind)
}

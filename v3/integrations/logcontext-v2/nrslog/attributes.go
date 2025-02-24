package nrslog

import (
	"log/slog"
	"maps"
	"strings"
)

type attributeCache struct {
	preCompiledAttributes map[string]interface{}
	prefix                string
}

func newAttributeCache() *attributeCache {
	return &attributeCache{
		preCompiledAttributes: make(map[string]interface{}),
		prefix:                "",
	}
}

func (c *attributeCache) clone() *attributeCache {
	return &attributeCache{
		preCompiledAttributes: maps.Clone(c.preCompiledAttributes),
		prefix:                c.prefix,
	}
}

func (c *attributeCache) copyPreCompiledAttributes() map[string]interface{} {
	return maps.Clone(c.preCompiledAttributes)
}

func (c *attributeCache) getPrefix() string {
	return c.prefix
}

// precompileGroup sets the group prefix for the cache created by a handler
// precompileGroup call. This is used to avoid re-computing the group prefix
// and should only ever be called on newly created caches and handlers.
func (c *attributeCache) precompileGroup(group string) {
	if c.prefix != "" {
		c.prefix += "."
	}
	c.prefix += group
}

// precompileAttributes appends attributes to the cache created by a handler
// WithAttrs call. This is used to avoid re-computing the with Attrs attributes
// and should only ever be called on newly created caches and handlers.
func (c *attributeCache) precompileAttributes(attrs []slog.Attr) {
	if len(attrs) == 0 {
		return
	}

	for _, a := range attrs {
		c.appendAttr(c.preCompiledAttributes, a, c.prefix)
	}
}

func (c *attributeCache) appendAttr(nrAttrs map[string]interface{}, a slog.Attr, groupPrefix string) {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return
	}

	// majority of runtime spent allocating and copying strings
	group := strings.Builder{}
	group.Grow(len(groupPrefix) + len(a.Key) + 1)
	group.WriteString(groupPrefix)

	if a.Key != "" {
		if group.Len() > 0 {
			group.WriteByte('.')
		}
		group.WriteString(a.Key)
	}

	key := group.String()

	// If the Attr is a group, append its attributes
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return
		}

		for _, ga := range attrs {
			c.appendAttr(nrAttrs, ga, key)
		}
		return
	}

	// attr is an attribute
	nrAttrs[key] = a.Value.Any()
}

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

func (c *attributeCache) getPreCompiledAttributes() map[string]interface{} {
	if c.preCompiledAttributes == nil {
		return make(map[string]interface{})
	}
	return maps.Clone(c.preCompiledAttributes)
}

func (c *attributeCache) getPrefix() string {
	return c.prefix
}

func (c *attributeCache) computePrecompiledAttributes(goas []groupOrAttrs) {
	if len(goas) == 0 {
		return
	}

	// if just one element, we can avoid allocation for the sting builder
	if len(goas) == 1 {
		if goas[0].group != "" {
			c.prefix = goas[0].group
		} else {
			attrs := make(map[string]interface{})
			for _, a := range goas[0].attrs {
				c.appendAttr(attrs, a, "")
			}
		}
		return
	}

	// string builder worth the pre-allocation cost
	groupPrefix := strings.Builder{}
	attrs := make(map[string]interface{})

	for _, goa := range goas {
		if goa.group != "" {
			if len(groupPrefix.String()) > 0 {
				groupPrefix.WriteByte('.')
			}
			groupPrefix.WriteString(goa.group)
		} else {
			for _, a := range goa.attrs {
				c.appendAttr(attrs, a, groupPrefix.String())
			}
		}
	}

	c.preCompiledAttributes = attrs
	c.prefix = groupPrefix.String()
}

func (c *attributeCache) appendAttr(nrAttrs map[string]interface{}, a slog.Attr, groupPrefix string) {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return
	}

	group := strings.Builder{}
	group.WriteString(groupPrefix)

	if group.Len() > 0 {
		group.WriteByte('.')
	}
	group.WriteString(a.Key)
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

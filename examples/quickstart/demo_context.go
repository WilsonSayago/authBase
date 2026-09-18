package main

import "context"

// demoContext is a minimal core.Context for the AuthorizeJWT quickstart.
type demoContext struct {
	header    string
	values    map[string]any
	req       context.Context
	aborted   bool
	abortCode int
	abortBody any
	nextCalls int
}

func (c *demoContext) GetHeader(key string) string {
	if key == "Authorization" {
		return c.header
	}
	return ""
}

func (c *demoContext) Set(key string, value interface{}) {
	if c.values == nil {
		c.values = make(map[string]any)
	}
	c.values[key] = value
}

func (c *demoContext) AbortWithStatusJSON(code int, jsonObj interface{}) {
	c.aborted = true
	c.abortCode = code
	c.abortBody = jsonObj
}

func (c *demoContext) Next() {
	c.nextCalls++
}

func (c *demoContext) Get(key string) (any, bool) {
	if c.values == nil {
		return nil, false
	}
	v, ok := c.values[key]
	return v, ok
}

func (c *demoContext) Status(code int) {}

func (c *demoContext) RequestContext() context.Context {
	if c.req == nil {
		return context.Background()
	}
	return c.req
}

package authctx

import (
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	userIDKey   = "userID"
	tokenExpKey = "tokenExp"
)

func SetUserID(c *app.RequestContext, id int64) {
	c.Set(userIDKey, id)
}

func UserID(c *app.RequestContext) (int64, bool) {
	v, ok := c.Get(userIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

func SetTokenExp(c *app.RequestContext, exp time.Time) {
	c.Set(tokenExpKey, exp)
}

func TokenExp(c *app.RequestContext) (time.Time, bool) {
	v, ok := c.Get(tokenExpKey)
	if !ok {
		return time.Time{}, false
	}
	exp, ok := v.(time.Time)
	return exp, ok
}

package authctx

import (
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestSetAndGetUserID(t *testing.T) {
	c := app.NewContext(16)

	id := int64(456)
	SetUserID(c, id)

	gotID, ok := UserID(c)
	if !ok {
		t.Fatal("expected to find user ID in context")
	}
	if gotID != id {
		t.Errorf("expected %d, got %d", id, gotID)
	}
}

func TestUserID_NotFound(t *testing.T) {
	c := app.NewContext(16)
	_, ok := UserID(c)
	if ok {
		t.Error("expected ok to be false for missing user ID")
	}
}

func TestUserID_WrongType(t *testing.T) {
	c := app.NewContext(16)
	c.Set(userIDKey, "not-an-int64")
	_, ok := UserID(c)
	if ok {
		t.Error("expected ok to be false for wrong type")
	}
}

func TestSetAndGetTokenExp(t *testing.T) {
	c := app.NewContext(16)

	exp := time.Now().Add(time.Hour)
	SetTokenExp(c, exp)

	gotExp, ok := TokenExp(c)
	if !ok {
		t.Fatal("expected to find token exp in context")
	}
	if !gotExp.Equal(exp) {
		t.Errorf("expected %v, got %v", exp, gotExp)
	}
}

func TestTokenExp_NotFound(t *testing.T) {
	c := app.NewContext(16)
	_, ok := TokenExp(c)
	if ok {
		t.Error("expected ok to be false for missing token exp")
	}
}

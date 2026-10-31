package bootstrap_test

import (
	"errors"
	"testing"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/bootstrap"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/config"
)

func TestNewRunner_APIIsNotImplemented(t *testing.T) {
	_, err := bootstrap.NewRunner(t.Context(), config.Config{Role: config.RoleAPI}, nil)
	if !errors.Is(err, bootstrap.ErrNotImplemented) {
		t.Errorf("err = %v", err)
	}
}

// Package theme synchronizes the web page's appearance with the native app.
package theme

import (
	"errors"
	"fmt"
)

// ApplyFunc applies a validated appearance mode.
type ApplyFunc func(mode string) error

// Service is bound to the page as `MiraTheme`.
type Service struct {
	apply ApplyFunc
}

// New creates the bound theme service.
func New(apply ApplyFunc) *Service {
	return &Service{apply: apply}
}

// SetTheme applies "system", "light", or "dark" to the native window.
func (s *Service) SetTheme(mode string) error {
	switch mode {
	case "system", "light", "dark":
	default:
		return fmt.Errorf("不支持的主题: %q", mode)
	}
	if s.apply == nil {
		return errors.New("原生主题服务未初始化")
	}
	return s.apply(mode)
}

package ux

import (
	"hillside/internal/ui"
	"testing"
)

func TestWelcomeUI(t *testing.T) {
	cfg := &ui.UIConfig{
		Theme: "default",
	}
	uiInstance := ui.NewUI(cfg)
	if uiInstance == nil {
		t.Fatal("Failed to create UI instance")
	}

	uiInstance.WelcomeUI.Init()
	if err := uiInstance.App.Run(); err != nil {
		t.Fatalf("Failed to run UI application: %v", err)
	}
}
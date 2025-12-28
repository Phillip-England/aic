package aic_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phillip-england/aic/src/aic"
)

func TestDollarToken_Path_ExpandsFilesRecursively(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}
	_ = os.MkdirAll(filepath.Join(td, "sub", "deep"), 0o755)
	_ = os.WriteFile(filepath.Join(td, "sub", "foo.txt"), []byte("bar\n"), 0o644)
	_ = os.WriteFile(filepath.Join(td, "sub", "deep", "x.txt"), []byte("x\n"), 0o644)

	pr := aic.NewPromptReader(`before $path("sub") after`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "FILE: "+filepath.Join(td, "sub", "foo.txt")) {
		t.Fatalf("expected output to include foo.txt, got:\n%s", out)
	}
	if !strings.Contains(out, "FILE: "+filepath.Join(td, "sub", "deep", "x.txt")) {
		t.Fatalf("expected output to include deep/x.txt, got:\n%s", out)
	}
}

func TestDollarToken_Jump_AddsPostActionAndRendersEmpty(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`a $jump(10,20) b`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "$jump(") {
		t.Fatalf("expected $jump to render empty, got:\n%s", out)
	}
	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	if pr.PostActions[0].Phase != aic.PostActionAfter ||
		pr.PostActions[0].Kind != aic.PostActionJump ||
		pr.PostActions[0].X != 10 ||
		pr.PostActions[0].Y != 20 {
		t.Fatalf("unexpected post action: %#v", pr.PostActions[0])
	}
}

func TestDollarToken_Click_AddsPostActionAndRendersEmpty(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`a $click("right") b`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "$click(") {
		t.Fatalf("expected $click to render empty, got:\n%s", out)
	}
	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	if pr.PostActions[0].Phase != aic.PostActionAfter ||
		pr.PostActions[0].Kind != aic.PostActionClick ||
		pr.PostActions[0].Button != "right" {
		t.Fatalf("unexpected post action: %#v", pr.PostActions[0])
	}
}

func TestDollarToken_Type_AddsPostActionAndRendersEmpty(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`a $type("v", ["CONTROL"]) b`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "$type(") {
		t.Fatalf("expected $type to render empty, got:\n%s", out)
	}
	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	pa := pr.PostActions[0]
	if pa.Phase != aic.PostActionAfter || pa.Kind != aic.PostActionType || pa.Text != "v" {
		t.Fatalf("unexpected post action: %#v", pa)
	}
	if len(pa.Mods) != 1 || pa.Mods[0] != "CONTROL" {
		t.Fatalf("unexpected mods: %#v", pa.Mods)
	}
}

func TestDollarToken_Type_ParsesDelayMs(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`$type("hello", ["SHIFT"], 15)`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	_, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	if pr.PostActions[0].DelayMs != 15 {
		t.Fatalf("expected DelayMs=15, got %d", pr.PostActions[0].DelayMs)
	}
}

func TestDollarToken_Sleep_AddsPostActionAndRendersEmpty_IntMs(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`a $sleep(250) b`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "$sleep(") {
		t.Fatalf("expected $sleep to render empty, got:\n%s", out)
	}
	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	pa := pr.PostActions[0]
	if pa.Phase != aic.PostActionAfter || pa.Kind != aic.PostActionSleep {
		t.Fatalf("unexpected post action: %#v", pa)
	}
	if pa.Sleep != 250*time.Millisecond {
		t.Fatalf("expected Sleep=250ms, got %v", pa.Sleep)
	}
}

func TestDollarToken_Sleep_AddsPostActionAndRendersEmpty_DurationString(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`$sleep("750ms")`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	_, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	if pr.PostActions[0].Kind != aic.PostActionSleep {
		t.Fatalf("expected PostActionSleep, got %#v", pr.PostActions[0])
	}
	if pr.PostActions[0].Sleep != 750*time.Millisecond {
		t.Fatalf("expected Sleep=750ms, got %v", pr.PostActions[0].Sleep)
	}
}

func TestDollarToken_Press_AddsPostActionAndRendersEmpty(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`a $press("BACKSPACE") b`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "$press(") {
		t.Fatalf("expected $press to render empty, got:\n%s", out)
	}
	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	pa := pr.PostActions[0]
	if pa.Phase != aic.PostActionAfter || pa.Kind != aic.PostActionPress {
		t.Fatalf("unexpected post action: %#v", pa)
	}
	if pa.Key != "BACKSPACE" {
		t.Fatalf("expected Key=BACKSPACE, got %q", pa.Key)
	}
}

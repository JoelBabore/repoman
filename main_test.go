package main

import "testing"

func TestGetExensionJson(t *testing.T) {
	want := "json"
	ext := GetExtension("file.json")
	if ext != want {
		t.Fatalf(`GetExtension("file.json") = %q,  want match for %q, nil`, ext, want)
	}
}

func TestGetExensionNoDot(t *testing.T) {
	want := "config"
	ext := GetExtension("config")
	if ext != want {
		t.Fatalf(`GetExtension("config") = %q,  want match for %q, nil`, ext, want)
	}
}

func TestGetExensionDotAtStart(t *testing.T) {
	want := ".config"
	ext := GetExtension(".config")
	if ext != want {
		t.Fatalf(`GetExtension(".config") = %q,  want match for %q, nil`, ext, want)
	}
}

func TestGetExensionSpace(t *testing.T) {
	want := "json"
	ext := GetExtension("Has A Space.json")
	if ext != want {
		t.Fatalf(`GetExtension("Has Space.json") = %q,  want match for %q, nil`, ext, want)
	}
}

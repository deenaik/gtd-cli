package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func Confirm(msg string) (bool, error) {
	var confirmed bool
	err := huh.NewConfirm().
		Title(msg).
		Value(&confirmed).
		Run()
	return confirmed, err
}

func SelectOne(title string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	var selected string
	err := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&selected).
		Run()
	return selected, err
}

func Input(title string, placeholder string) (string, error) {
	var value string
	err := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Value(&value).
		Run()
	return value, err
}

func TextArea(title string, placeholder string) (string, error) {
	var value string
	err := huh.NewText().
		Title(title).
		Placeholder(placeholder).
		Value(&value).
		Run()
	return value, err
}

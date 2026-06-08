package main

import (
	"fmt"
	"strings"
)

var (
	validAspectRatios = map[string]bool{
		"1:1": true, "16:9": true, "9:16": true, "4:3": true, "3:4": true,
		"3:2": true, "2:3": true, "2:1": true, "1:2": true,
		"19.5:9": true, "9:19.5": true, "20:9": true, "9:20": true, "auto": true,
	}
	validResolutions  = map[string]bool{"1k": true, "2k": true}
	validServiceTiers = map[string]bool{"default": true, "priority": true}
	validModels       = map[string]bool{
		"grok-imagine-image-quality": true,
		"grok-imagine-image":         true,
	}
)

func validatePrompt(prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt is required and cannot be empty")
	}
	return nil
}

func validateImagePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("imagePath is required and cannot be empty")
	}
	return nil
}

func validateOptionalEnum(value *string, field string, allowed map[string]bool) error {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	if !allowed[v] {
		return fmt.Errorf("invalid %s: %q", field, v)
	}
	return nil
}

func validateNumberOfImages(n *int) error {
	if n == nil {
		return nil
	}
	if *n < 1 || *n > 10 {
		return fmt.Errorf("numberOfImages must be between 1 and 10, got %d", *n)
	}
	return nil
}

func validateGenerateImageArgs(prompt string, model, aspectRatio, resolution, serviceTier *string, numberOfImages *int) error {
	if err := validatePrompt(prompt); err != nil {
		return err
	}
	if err := validateOptionalEnum(model, "model", validModels); err != nil {
		return err
	}
	if err := validateOptionalEnum(aspectRatio, "aspectRatio", validAspectRatios); err != nil {
		return err
	}
	if err := validateOptionalEnum(resolution, "resolution", validResolutions); err != nil {
		return err
	}
	if err := validateOptionalEnum(serviceTier, "serviceTier", validServiceTiers); err != nil {
		return err
	}
	return validateNumberOfImages(numberOfImages)
}

func validateEditImageArgs(imagePath, prompt string, model, aspectRatio, resolution, serviceTier *string, numberOfImages *int) error {
	if err := validateImagePath(imagePath); err != nil {
		return err
	}
	return validateGenerateImageArgs(prompt, model, aspectRatio, resolution, serviceTier, numberOfImages)
}
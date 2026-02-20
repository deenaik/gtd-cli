package ai

import (
	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// SmartCaptureSchema returns the JSON schema for structured task capture output.
func SmartCaptureSchema() *openai.ChatCompletionResponseFormatJSONSchema {
	return &openai.ChatCompletionResponseFormatJSONSchema{
		Name:        "smart_capture",
		Description: "Structured task parsed from natural language input",
		Strict:      true,
		Schema: &jsonschema.Definition{
			Type:                 jsonschema.Object,
			AdditionalProperties: false,
			Required:             []string{"title", "description", "project", "context", "priority", "due_date", "energy", "time_estimate"},
			Properties: map[string]jsonschema.Definition{
				"title": {
					Type:        jsonschema.String,
					Description: "Clear, actionable task title starting with a verb",
				},
				"description": {
					Type:        jsonschema.String,
					Description: "Additional details or notes; empty string if none",
				},
				"project": {
					Type:        jsonschema.String,
					Description: "Project name if applicable; empty string if none",
				},
				"context": {
					Type:        jsonschema.String,
					Description: "GTD context (e.g., @office, @home, @phone, @computer, @errands, @anywhere)",
				},
				"priority": {
					Type:        jsonschema.Integer,
					Description: "Priority level: 1=low, 2=medium, 3=high, 4=urgent",
					Enum:        []string{"1", "2", "3", "4"},
				},
				"due_date": {
					Type:        jsonschema.String,
					Description: "ISO 8601 date (YYYY-MM-DD) if a deadline exists; empty string if none",
				},
				"energy": {
					Type:        jsonschema.String,
					Description: "Energy level required for the task",
					Enum:        []string{"low", "medium", "high"},
				},
				"time_estimate": {
					Type:        jsonschema.Integer,
					Description: "Estimated minutes to complete the task",
				},
			},
		},
	}
}

// CategorizeSchema returns the JSON schema for GTD inbox item categorization.
func CategorizeSchema() *openai.ChatCompletionResponseFormatJSONSchema {
	return &openai.ChatCompletionResponseFormatJSONSchema{
		Name:        "categorize_item",
		Description: "GTD categorization of an inbox item",
		Strict:      true,
		Schema: &jsonschema.Definition{
			Type:                 jsonschema.Object,
			AdditionalProperties: false,
			Required:             []string{"category", "confidence", "reasoning", "suggested_context", "suggested_project", "priority", "energy"},
			Properties: map[string]jsonschema.Definition{
				"category": {
					Type:        jsonschema.String,
					Description: "GTD category for the item",
					Enum:        []string{"next_action", "waiting_for", "someday_maybe", "reference", "trash"},
				},
				"confidence": {
					Type:        jsonschema.Number,
					Description: "Confidence score between 0.0 and 1.0",
				},
				"reasoning": {
					Type:        jsonschema.String,
					Description: "Brief explanation of why this category was chosen",
				},
				"suggested_context": {
					Type:        jsonschema.String,
					Description: "Suggested GTD context (e.g., @office, @home); empty if not applicable",
				},
				"suggested_project": {
					Type:        jsonschema.String,
					Description: "Suggested project name if applicable; empty string if standalone",
				},
				"priority": {
					Type:        jsonschema.Integer,
					Description: "Suggested priority: 0=none, 1=low, 2=medium, 3=high, 4=urgent",
				},
				"energy": {
					Type:        jsonschema.String,
					Description: "Suggested energy level required",
					Enum:        []string{"low", "medium", "high"},
				},
			},
		},
	}
}

// EmailClassifySchema returns the JSON schema for email importance classification.
// The schema wraps the array in an object because structured outputs require a
// top-level object.
func EmailClassifySchema() *openai.ChatCompletionResponseFormatJSONSchema {
	return &openai.ChatCompletionResponseFormatJSONSchema{
		Name:        "email_classify",
		Description: "Email importance classification results",
		Strict:      true,
		Schema: &jsonschema.Definition{
			Type:                 jsonschema.Object,
			AdditionalProperties: false,
			Required:             []string{"classifications"},
			Properties: map[string]jsonschema.Definition{
				"classifications": {
					Type:        jsonschema.Array,
					Description: "List of email classifications",
					Items: &jsonschema.Definition{
						Type:                 jsonschema.Object,
						AdditionalProperties: false,
						Required:             []string{"email_id", "importance", "reason"},
						Properties: map[string]jsonschema.Definition{
							"email_id": {
								Type:        jsonschema.String,
								Description: "Identifier of the email being classified",
							},
							"importance": {
								Type:        jsonschema.String,
								Description: "Importance level of the email",
								Enum:        []string{"high", "medium", "low"},
							},
							"reason": {
								Type:        jsonschema.String,
								Description: "Brief explanation of the classification",
							},
						},
					},
				},
			},
		},
	}
}

// ScheduleSuggestionSchema returns the JSON schema for smart scheduling suggestions.
func ScheduleSuggestionSchema() *openai.ChatCompletionResponseFormatJSONSchema {
	return &openai.ChatCompletionResponseFormatJSONSchema{
		Name:        "schedule_suggestion",
		Description: "Suggested time slots for scheduling a task",
		Strict:      true,
		Schema: &jsonschema.Definition{
			Type:                 jsonschema.Object,
			AdditionalProperties: false,
			Required:             []string{"suggestions"},
			Properties: map[string]jsonschema.Definition{
				"suggestions": {
					Type:        jsonschema.Array,
					Description: "List of 1-3 suggested time slots",
					Items: &jsonschema.Definition{
						Type:                 jsonschema.Object,
						AdditionalProperties: false,
						Required:             []string{"start", "end", "reasoning"},
						Properties: map[string]jsonschema.Definition{
							"start": {
								Type:        jsonschema.String,
								Description: "ISO 8601 datetime for suggested start (YYYY-MM-DDTHH:MM:SS)",
							},
							"end": {
								Type:        jsonschema.String,
								Description: "ISO 8601 datetime for suggested end (YYYY-MM-DDTHH:MM:SS)",
							},
							"reasoning": {
								Type:        jsonschema.String,
								Description: "Brief explanation of why this time slot is recommended",
							},
						},
					},
				},
			},
		},
	}
}

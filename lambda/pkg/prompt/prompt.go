package prompt

// Prompt represents the structure of our prompt
type Prompt struct {
	Namespace           string   `json:"namespace,omitempty" validate:"required" binding:"required"`
	Team                string   `json:"team,omitempty" validate:"required" binding:"required"`
	Name                string   `json:"name" validate:"required" binding:"required"`
	PromptText          string   `json:"text" validate:"required" binding:"required"`
	InterpolationValues []string `json:"interpolation_values,omitempty"`
	Description         string   `json:"description"`
	Tags                []string `json:"tags,omitempty"`
	Meta                struct {
		Authors []string `json:"authors,omitempty" validate:"required" binding:"required"`
	} `json:"meta,omitempty"`
	Version string `json:"version" validate:"required" binding:"required" example:"1.0.0"`
}

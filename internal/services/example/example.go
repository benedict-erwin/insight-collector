package example

import (
	"fmt"

	"github.com/benedict-erwin/insight-collector/internal/entities/example"
)

// ExampleService demonstrates business logic with validated input
func ProcessExample(req *example.ExampleRequest) (map[string]interface{}, error) {
	// Business logic here - validation already done by echo.Validator
	result := map[string]interface{}{
		"message": fmt.Sprintf("Hello %s, processing %s action", req.Name, req.Action),
		"user_info": map[string]interface{}{
			"name":  req.Name,
			"email": req.Email,
			"age":   req.Age,
		},
		"action_performed": req.Action,
	}

	// Add optional field if provided
	if req.Optional != "" {
		result["optional_data"] = req.Optional
	}

	return result, nil
}

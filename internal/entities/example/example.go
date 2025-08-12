package example

// ExampleRequest demonstrates validation tags usage
type ExampleRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"required,min=1,max=120"`
	Action   string `json:"action" validate:"required,oneof=create update delete"`
	Optional string `json:"optional" validate:"omitempty,min=5"`
}
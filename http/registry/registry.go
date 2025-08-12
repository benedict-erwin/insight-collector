package registry

import (
	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

type SetupFunc func(g *echo.Group)

var versionRegistry = make(map[string][]SetupFunc)

// Register router setup function for specific API version
func Register(version string, setup SetupFunc) {
	logger.WithScope("RegistryRegister").Info().Str("version", version).Msg("Registering router for version")
	versionRegistry[version] = append(versionRegistry[version], setup)
}

// SetupAllRoutes applies all registered routes
func SetupAllRoutes(e *echo.Echo) {
	// Initialize validator
	setupValidator(e)

	// Setup logger scope
	log := logger.WithScope("SetupAllRoutes")

	// Register routes
	if len(versionRegistry) == 0 {
		log.Warn().Msg("No routes registered in versionRegistry")
		return
	}
	for version, setups := range versionRegistry {
		log.Info().Str("version", version).Int("routes", len(setups)).Msg("Setting up version group")
		// Create version group
		g := e.Group("/" + version)
		for i, setup := range setups {
			log.Info().Str("version", version).Int("route_index", i).Msg("Applying route setup")
			setup(g)
		}
	}
}

// setupValidator configures request validation using go-playground/validator
func setupValidator(e *echo.Echo) {
	v := validator.New()
	e.Validator = &CustomValidator{validator: v}
	logger.WithScope("RegistrysetupValidator").Info().Msg("Validator setup completed")
}

type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates struct fields using validator tags
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

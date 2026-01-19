package health

// healthCheckInput represents the input for health check endpoint
type healthCheckInput struct{}

// healthCheckOutput represents the output for health check endpoint
type healthCheckOutput struct {
	Body HealthCheckResponse
}

// HealthCheckResponse represents the health check response
type HealthCheckResponse struct {
	Status string `json:"status" example:"OK" doc:"Health status of the service"`
}

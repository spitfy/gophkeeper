package health

// Input represents the input for health check endpoint
type Input struct{}

// Output represents the output for health check endpoint
type Output struct {
	Body Response
}

// Response represents the health check response
type Response struct {
	Status string `json:"status" example:"OK" doc:"Health status of the service"`
}

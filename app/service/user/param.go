package user

import "time"

type ImportOptions struct {
	MaxWorkers int
	QueueSize  int
}

type ImportSummary struct {
	Total      int           `json:"total"`
	Successful int           `json:"successful"`
	Failed     int           `json:"failed"`
	Duration   time.Duration `json:"duration"`
}

type GetUserResponse struct {
	User User `json:"user"`
}

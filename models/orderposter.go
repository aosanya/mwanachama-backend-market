package models

type OrderPoster struct {
	PosterID   string `json:"poster_id"`
	PosterName string `json:"poster_name"`
	Source     string `json:"source"`
}

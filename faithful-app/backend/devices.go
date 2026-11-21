package main

type device struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Buttons []bool    `json:"buttons"`
	Axes    []float64 `json:"axes"`
	POV     int       `json:"pov"`
}

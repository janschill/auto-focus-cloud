package models

const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
)

type License struct {
	Key     string
	Version string
	Status  string
}

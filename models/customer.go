package models

type Customer struct {
	Id       string
	Email    string
	Licenses []License
}

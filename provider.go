package main

type Provider struct {
	Id  string
	Url string
}

type IProvider interface {
	Check(*Availability)
	GetId() string
	GetUrl() string
}

func (provider Provider) GetId() string {
	return provider.Id
}

func (provider Provider) GetUrl() string {
	return provider.Url
}

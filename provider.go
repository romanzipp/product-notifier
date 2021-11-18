package main

import "errors"

type Provider struct {
	Id  string
	Url string
}

type IProvider interface {
	GetId() string
	GetUrl() string
	GetAvailableSizes() ([]string, error)
}

func (provider Provider) GetId() string {
	return provider.Id
}

func (provider Provider) GetUrl() string {
	return provider.Url
}

func GetProviderById(id string, url string) (IProvider, error) {
	switch id {
	case ProviderNike:
		return Nike{
			Provider{
				Id:  ProviderNike,
				Url: url,
			},
		}, nil
	case ProviderZalando:
		return Zalando{
			Provider{
				Id:  ProviderZalando,
				Url: url,
			},
		}, nil
	}

	return nil, errors.New("unknown provider")
}

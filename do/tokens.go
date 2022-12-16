package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// TokensService is an interface for managing DigitalOcean's API tokens.
type TokensService interface {
	List() ([]Token, error)
	Get(int) (*Token, error)
	Create(*godo.TokenCreateRequest) (*Token, error)
	Update(int, *godo.TokenUpdateRequest) (*Token, error)
	Revoke(int) error
	ListScopes(string) ([]TokenScope, error)
}

// Token is a wrapper for godo.Token.
type Token struct {
	*godo.Token
}

// TokenScope is a wrapper for godo.TokenScope.
type TokenScope struct {
	*godo.TokenScope
}

type tokensService struct {
	client *godo.Client
}

// NewTokensService builds a new TokensService instance.
func NewTokensService(godoClient *godo.Client) TokensService {
	return &tokensService{
		client: godoClient,
	}
}

func (t *tokensService) List() ([]Token, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := t.client.Tokens.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]Token, len(si))
	for i := range si {
		token := si[i].(godo.Token)
		list[i] = Token{Token: &token}
	}
	return list, nil
}

func (t *tokensService) Get(id int) (*Token, error) {
	token, _, err := t.client.Tokens.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	return &Token{Token: token}, nil
}

func (t *tokensService) Create(req *godo.TokenCreateRequest) (*Token, error) {
	token, _, err := t.client.Tokens.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	return &Token{Token: token}, nil
}

func (t *tokensService) Update(id int, req *godo.TokenUpdateRequest) (*Token, error) {
	token, _, err := t.client.Tokens.Update(context.TODO(), id, req)
	if err != nil {
		return nil, err
	}

	return &Token{Token: token}, nil
}

func (t *tokensService) Revoke(id int) error {
	_, err := t.client.Tokens.Revoke(context.TODO(), id)

	return err
}

func (t *tokensService) ListScopes(namespace string) ([]TokenScope, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		var (
			list []godo.TokenScope
			resp *godo.Response
			err  error
		)

		if namespace != "" {
			list, resp, err = t.client.Tokens.ListScopesByNamespace(context.TODO(), namespace, opt)
			if err != nil {
				return nil, nil, err
			}
		} else {
			list, resp, err = t.client.Tokens.ListScopes(context.TODO(), opt)
			if err != nil {
				return nil, nil, err
			}
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]TokenScope, len(si))
	for i := range si {
		scope := si[i].(godo.TokenScope)
		list[i] = TokenScope{TokenScope: &scope}
	}

	return list, nil
}

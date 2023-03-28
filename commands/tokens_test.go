package commands

import (
	"bytes"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testTokenID = "123"

	testToken = do.Token{
		Token: &godo.Token{
			ID:            123,
			Name:          "droplets-read-token",
			Scopes:        []string{"droplet:read"},
			CreatedAt:     time.Date(2022, time.December, 5, 20, 32, 15, 0, time.UTC),
			LastUsedAt:    time.Date(2022, time.December, 5, 20, 32, 15, 0, time.UTC).Format("2006-01-02"),
			ExpirySeconds: godo.PtrTo(604800),
		},
	}

	testTokenList = []do.Token{
		{Token: testToken.Token},
	}

	testTokenCreateResp = do.Token{
		Token: &godo.Token{
			ID:            123,
			Name:          "droplets-read-token",
			Scopes:        []string{"droplet:read"},
			CreatedAt:     time.Date(2022, time.December, 5, 20, 32, 15, 0, time.UTC),
			LastUsedAt:    time.Date(2022, time.December, 5, 20, 32, 15, 0, time.UTC).Format("2006-01-02"),
			ExpirySeconds: godo.PtrTo(3600),
			AccessToken:   "shhhhhhhimsecret",
		},
	}

	testTokenScopesList = []do.TokenScope{
		{
			TokenScope: &godo.TokenScope{Name: "account:read"},
		},
		{
			TokenScope: &godo.TokenScope{Name: "droplet:read"},
		},
	}
)

func TestTokensCommand(t *testing.T) {
	cmd := Tokens()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "revoke", "get", "list", "update", "list-scopes")
}

func TestTokensGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tokens.EXPECT().Get(testToken.ID).Return(&testToken, nil)

		config.Args = append(config.Args, testTokenID)

		err := RunTokenGet(config)
		assert.NoError(t, err)
	})
}

func TestTokensGetByName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tokens.EXPECT().List().Return(testTokenList, nil)

		config.Args = append(config.Args, testToken.Name)

		err := RunTokenGet(config)
		assert.NoError(t, err)
	})
}

func TestTokensGetFormatted(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.tokens.EXPECT().Get(testToken.ID).Return(&testToken, nil)

		config.Args = append(config.Args, testTokenID)
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)
		config.Doit.Set(config.NS, doctl.ArgFormat, "Name")

		err := RunTokenGet(config)
		assert.NoError(t, err)
		assert.Equal(t, testToken.Name+"\n", buf.String())
	})
}

func TestTokensGetMissingRequiredArguments(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunTokenGet(config)
		assert.Error(t, err)
	})
}

func TestTokensList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tokens.EXPECT().List().Return(testTokenList, nil)

		err := RunTokenList(config)
		assert.NoError(t, err)
	})
}

func TestTokensListFormatted(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.tokens.EXPECT().List().Return(testTokenList, nil)

		config.Args = append(config.Args, testTokenID)
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)
		config.Doit.Set(config.NS, doctl.ArgFormat, "Name")

		err := RunTokenList(config)
		assert.NoError(t, err)
		assert.Equal(t, testToken.Name+"\n", buf.String())
	})
}

func TestTokensCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		createReq := &godo.TokenCreateRequest{
			Name:          testToken.Name,
			ExpirySeconds: godo.PtrTo(3600),
			Scopes:        testToken.Scopes,
		}
		tm.tokens.EXPECT().Create(createReq).Return(&testTokenCreateResp, nil)

		config.Args = append(config.Args, testToken.Name)
		config.Doit.Set(config.NS, doctl.ArgTokenExpirySeconds, "3600")
		config.Doit.Set(config.NS, doctl.ArgTokenScopes, "droplet:read")

		err := RunTokenCreate(config)
		assert.NoError(t, err)
	})
}

func TestTokensCreateWithDuration(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		createReq := &godo.TokenCreateRequest{
			Name:          testToken.Name,
			ExpirySeconds: godo.PtrTo(86400),
			Scopes:        testToken.Scopes,
		}
		tm.tokens.EXPECT().Create(createReq).Return(&testTokenCreateResp, nil)

		config.Args = append(config.Args, testToken.Name)
		config.Doit.Set(config.NS, doctl.ArgTokenExpiresIn, "24h")
		config.Doit.Set(config.NS, doctl.ArgTokenScopes, "droplet:read")

		err := RunTokenCreate(config)
		assert.NoError(t, err)
	})
}

func TestTokensCreateNoExpiration(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		createReq := &godo.TokenCreateRequest{
			Name:   testToken.Name,
			Scopes: testToken.Scopes,
		}
		tm.tokens.EXPECT().Create(createReq).Return(&testTokenCreateResp, nil)

		config.Args = append(config.Args, testToken.Name)
		config.Doit.Set(config.NS, doctl.ArgTokenScopes, "droplet:read")

		err := RunTokenCreate(config)
		assert.NoError(t, err)
	})
}

func TestTokensCreateHasAccessToken(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		createReq := &godo.TokenCreateRequest{
			Name:          testToken.Name,
			ExpirySeconds: godo.PtrTo(3600),
			Scopes:        testToken.Scopes,
		}
		tm.tokens.EXPECT().Create(createReq).Return(&testTokenCreateResp, nil)

		config.Args = append(config.Args, testToken.Name)
		config.Doit.Set(config.NS, doctl.ArgTokenExpirySeconds, "3600")
		config.Doit.Set(config.NS, doctl.ArgTokenScopes, "droplet:read")
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)
		config.Doit.Set(config.NS, doctl.ArgFormat, "AccessToken")

		err := RunTokenCreate(config)
		assert.NoError(t, err)
		assert.Equal(t, testTokenCreateResp.AccessToken+"\n", buf.String())
	})
}

func TestTokensCreateMissingRequiredArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgTokenExpirySeconds, "3600")
		config.Doit.Set(config.NS, doctl.ArgTokenScopes, "droplet:read")

		err := RunTokenCreate(config)
		assert.Error(t, err)
	})
}

func TestTokensCreateMutuallyExclusiveArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgTokenExpirySeconds, "3600")
		config.Doit.Set(config.NS, doctl.ArgTokenExpiresIn, "3600s")
		config.Doit.Set(config.NS, doctl.ArgTokenScopes, "droplet:read")

		config.Args = append(config.Args, testToken.Name)

		err := RunTokenCreate(config)
		assert.Error(t, err)
	})
}

func TestTokensRevoke(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tokens.EXPECT().Revoke(testToken.ID).Return(nil)

		config.Args = append(config.Args, testTokenID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunTokenRevoke(config)
		assert.NoError(t, err)
	})
}

func TestTokensRevokeByName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tokens.EXPECT().List().Return(testTokenList, nil)
		tm.tokens.EXPECT().Revoke(testToken.ID).Return(nil)

		config.Args = append(config.Args, testToken.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunTokenRevoke(config)
		assert.NoError(t, err)
	})
}

func TestTokensRevokeMissingRequiredArguments(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunTokenRevoke(config)
		assert.Error(t, err)
	})
}

func TestTokensUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		updateReq := &godo.TokenUpdateRequest{
			Name:   "new-name",
			Scopes: []string{"droplet:delete"},
		}
		tm.tokens.EXPECT().Update(testToken.ID, updateReq).Return(&testToken, nil)

		config.Args = append(config.Args, testTokenID)
		config.Doit.Set(config.NS, doctl.ArgTokenUpdatedName, "new-name")
		config.Doit.Set(config.NS, doctl.ArgTokenScopes, "droplet:delete")

		err := RunTokenUpdate(config)
		assert.NoError(t, err)
	})
}

func TestTokensUpdateByName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		updateReq := &godo.TokenUpdateRequest{
			Name:   "new-name",
			Scopes: []string{"droplet:delete"},
		}
		tm.tokens.EXPECT().List().Return(testTokenList, nil)
		tm.tokens.EXPECT().Update(testToken.ID, updateReq).Return(&testToken, nil)

		config.Args = append(config.Args, testToken.Name)
		config.Doit.Set(config.NS, doctl.ArgTokenUpdatedName, "new-name")
		config.Doit.Set(config.NS, doctl.ArgTokenScopes, "droplet:delete")

		err := RunTokenUpdate(config)
		assert.NoError(t, err)
	})
}

func TestTokensUpdateMissingArgs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testTokenID)

		err := RunTokenUpdate(config)
		assert.Error(t, err)
	})
}

func TestTokensListScopes(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tokens.EXPECT().ListScopes("").Return(testTokenScopesList, nil)

		err := RunTokenListScopes(config)
		assert.NoError(t, err)
	})
}

func TestTokensListScopesByNamespace(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tokens.EXPECT().ListScopes("droplet").Return(testTokenScopesList, nil)

		config.Doit.Set(config.NS, doctl.ArgTokenScopeNamespace, "droplet")

		err := RunTokenListScopes(config)
		assert.NoError(t, err)
	})
}

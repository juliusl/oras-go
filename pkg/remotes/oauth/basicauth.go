package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

func NewBasicAuthTokenSource(ctx context.Context, namespace, username, password string, scopes string) oauth2.TokenSource {
	src := &basicAuthTokenSource{
		tokenFunc: func() (*oauth2.Token, error) {
			req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/oauth2/token?service=%s&scope=%s", namespace, namespace, scopes), nil)
			if err != nil {
				return nil, err
			}
			req.SetBasicAuth(username, password)

			c, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
			if !ok {
				c = http.DefaultClient
			}

			resp, err := c.Do(req)
			if err != nil {
				return nil, err
			}

			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return nil, fmt.Errorf("could not get access token")
			}

			token := &oauth2.Token{}
			if err := json.NewDecoder(resp.Body).Decode(token); err != nil {
				return nil, err
			}

			return token, nil
		},
	}

	token, err := src.Token()
	if err != nil {
		return nil
	}

	return oauth2.ReuseTokenSource(token, src)
}

type basicAuthTokenSource struct {
	tokenFunc TokenFunc
}

type TokenFunc = func() (*oauth2.Token, error)

func (b basicAuthTokenSource) Token() (*oauth2.Token, error) {
	return b.tokenFunc()
}

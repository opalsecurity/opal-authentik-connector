package authentik

import (
	"context"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	authentik "goauthentik.io/api/v3"
)

const AuthentikTokenEnvKey = "AUTHENTIK_TOKEN"

const PageQueryParam = "cursor"

const DefaultPageSize = 100

// Default authentication strategy, look for token in environment variables
func getTokenFromEnv() (token string, ok bool) {
	if _, hasEnv := os.LookupEnv(AuthentikTokenEnvKey); !hasEnv {
		return "", false
	}

	token = os.Getenv(AuthentikTokenEnvKey)
	return token, true
}

func getToken() (token string, ok bool) {
	// Currently only support getting auth token from env, can add more in the future
	getTokenFuncs := []func() (token string, ok bool){
		getTokenFromEnv,
	}

	for _, getTokenFunc := range getTokenFuncs {
		token, ok = getTokenFunc()
		if ok {
			return token, true
		}
	}

	return "", false
}

type AuthentikClient struct {
	token  string
	client *authentik.APIClient
}

func NewAuthentikClient() (*AuthentikClient, error) {
	token, ok := getToken()
	if !ok {
		return nil, errors.Errorf("Unable to find authentik token!")
	}

	return &AuthentikClient{
		token:  token,
		client: authentik.NewAPIClient(authentik.NewConfiguration()),
	}, nil
}

func (c *AuthentikClient) PaginatedListUsers(ctx *gin.Context) (users []authentik.User, nextCursor string, err error) {
	page, err := getPageFromCtx(ctx)
	if err != nil {
		return nil, "", errors.Wrap(err, "Encountered error while getting page number from request!")
	}

	ctxWithAuth := context.WithValue(ctx, authentik.ContextAccessToken, c.token)
	paginatedUsers, _, err := c.client.CoreApi.CoreUsersList(ctxWithAuth).Page(page).PageSize(DefaultPageSize).Execute()
	if err != nil {
		return nil, "", errors.Wrap(err, "Failed to list users from Authentik!")
	}

	return paginatedUsers.Results, getNextCursorFromPagination(paginatedUsers.Pagination), nil
}

func getNextCursorFromPagination(pagination authentik.Pagination) string {
	// If on last page, return empty next cursor, which means all resources have been fetched
	if pagination.TotalPages == pagination.Current {
		return ""
	}

	return strconv.FormatFloat(float64(pagination.Next), 'g', 1, 32)
}

func getPageFromCtx(ctx *gin.Context) (int32, error) {
	page, err := strconv.Atoi(ctx.DefaultQuery(PageQueryParam, "0"))
	if err != nil {
		return -1, err
	}

	return int32(page), nil
}

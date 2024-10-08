package openapi

import (
	"context"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	authentik "goauthentik.io/api/v3"
)

const (
	AuthentikTokenEnvKey  = "AUTHENTIK_TOKEN"
	AuthentikHostEnvKey   = "AUTHENTIK_HOST"
	AuthentikSchemeEnvKey = "AUTHENTIK_SCHEME"
)

const PageQueryParam = "cursor"

const DefaultPageSize = 100

type ClientError struct {
	innerError error
	StatusCode int
	Message    string
}

func (e *ClientError) Error() string {
	return "error: " + e.Message + " due to: " + e.innerError.Error()
}

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

	configuration := authentik.NewConfiguration()
	configuration.Host = os.Getenv(AuthentikHostEnvKey)
	configuration.Scheme = os.Getenv(AuthentikSchemeEnvKey)

	return &AuthentikClient{
		token:  token,
		client: authentik.NewAPIClient(configuration),
	}, nil
}

func (c *AuthentikClient) PaginatedListUsers(ctx *gin.Context) (users []authentik.User, nextCursor string, err error) {
	page, err := getPageFromCtx(ctx)
	if err != nil {
		return nil, "", errors.Wrap(err, "Encountered error while getting page number from request!")
	}

	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	paginatedUsers, resp, err := c.client.CoreApi.CoreUsersList(ctxWithAuth).Page(page).PageSize(DefaultPageSize).Execute()
	if err != nil {
		return nil, "", &ClientError{StatusCode: resp.StatusCode, Message: "Failed to list users from Authentik!", innerError: err}
	}

	return paginatedUsers.Results, getNextCursorFromPagination(paginatedUsers.Pagination), nil
}

func (c *AuthentikClient) addAuthTokenToCtx(ctx *gin.Context) context.Context {
	return context.WithValue(ctx, authentik.ContextAccessToken, c.token)
}

func (c *AuthentikClient) PaginatedListGroups(ctx *gin.Context) (groups []authentik.Group, nextCursor string, err error) {
	page, err := getPageFromCtx(ctx)
	if err != nil {
		return nil, "", errors.Wrap(err, "Encountered error while getting page number from request!")
	}

	ctxWithAuth := context.WithValue(ctx, authentik.ContextAccessToken, c.token)
	paginatedGroups, _, err := c.client.CoreApi.CoreGroupsList(ctxWithAuth).Page(page).PageSize(DefaultPageSize).Execute()
	if err != nil {
		return nil, "", errors.Wrap(err, "Failed to list groups from Authentik!")
	}

	return paginatedGroups.Results, getNextCursorFromPagination(paginatedGroups.Pagination), nil
}

func getNextCursorFromPagination(pagination authentik.Pagination) string {
	// If on last page, return empty next cursor, which means all resources have been fetched
	if pagination.TotalPages == pagination.Current {
		return ""
	}

	return strconv.FormatFloat(float64(pagination.Next), 'g', 1, 32)
}

func getPageFromCtx(ctx *gin.Context) (int32, error) {
	page, err := strconv.Atoi(ctx.DefaultQuery(PageQueryParam, "1"))
	if err != nil {
		return -1, err
	}

	return int32(page), nil
}

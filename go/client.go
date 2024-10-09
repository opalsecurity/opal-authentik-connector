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
		return nil, "", &ClientError{StatusCode: resp.StatusCode, Message: "failed to list users from Authentik", innerError: err}
	}

	return paginatedUsers.Results, getNextCursorFromPagination(paginatedUsers.Pagination), nil
}

func (c *AuthentikClient) PaginatedListGroups(ctx *gin.Context) (groups []authentik.Group, nextCursor string, err error) {
	page, err := getPageFromCtx(ctx)
	if err != nil {
		return nil, "", errors.Wrap(err, "Encountered error while getting page number from request!")
	}

	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	paginatedGroups, resp, err := c.client.CoreApi.CoreGroupsList(ctxWithAuth).Page(page).PageSize(DefaultPageSize).Execute()
	if err != nil {
		return nil, "", &ClientError{StatusCode: resp.StatusCode, Message: "failed to list groups from Authentik", innerError: err}
	}

	return paginatedGroups.Results, getNextCursorFromPagination(paginatedGroups.Pagination), nil
}

func (c *AuthentikClient) GetGroupUsers(ctx *gin.Context, groupID string) (members []authentik.GroupMember, err error) {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	group, resp, err := c.client.CoreApi.CoreGroupsRetrieve(ctxWithAuth, groupID).IncludeUsers(true).Execute()
	if err != nil {
		return nil, &ClientError{StatusCode: resp.StatusCode, Message: "failed to get users for group from Authentik", innerError: err}
	}

	return group.UsersObj, nil
}

func (c *AuthentikClient) GetGroup(ctx *gin.Context, groupID string) (group *authentik.Group, err error) {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	group, resp, err := c.client.CoreApi.CoreGroupsRetrieve(ctxWithAuth, groupID).IncludeUsers(false).Execute()
	if err != nil {
		return nil, &ClientError{StatusCode: resp.StatusCode, Message: "failed to get group from authentik", innerError: err}
	}

	return group, nil
}

func (c *AuthentikClient) AddUserToGroup(ctx *gin.Context, groupID string, userID string) error {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	// User ID provided to opal is user's primary key
	userPK, err := strconv.Atoi(userID)
	if err != nil {
		return err
	}
	userAccountRequest := authentik.NewUserAccountRequest(int32(userPK))

	resp, err := c.client.CoreApi.CoreGroupsAddUserCreate(ctxWithAuth, groupID).UserAccountRequest(*userAccountRequest).Execute()
	if err != nil {
		return &ClientError{StatusCode: resp.StatusCode, Message: "Failed to add user to group!", innerError: err}
	}

	return err
}

func (c *AuthentikClient) RemoveUserFromGroup(ctx *gin.Context, groupID string, userID string) error {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	// User ID provided to opal is user's primary key
	userPK, err := strconv.Atoi(userID)
	if err != nil {
		return err
	}
	userAccountRequest := authentik.NewUserAccountRequest(int32(userPK))

	resp, err := c.client.CoreApi.CoreGroupsRemoveUserCreate(ctxWithAuth, groupID).UserAccountRequest(*userAccountRequest).Execute()
	if err != nil {
		return &ClientError{StatusCode: resp.StatusCode, Message: "Failed to add user to group!", innerError: err}
	}

	return err
}

func (c *AuthentikClient) addAuthTokenToCtx(ctx *gin.Context) context.Context {
	return context.WithValue(ctx, authentik.ContextAccessToken, c.token)
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

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
	CFAccessClientID      = "CLOUDFLARE_ACCESS_CLIENT_ID"
	CFAccessClientSecret  = "CLOUDFLARE_ACCESS_CLIENT_SECRET"
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

	// Add Cloudflare Access token headers to the default headers
	clientID := os.Getenv("CF_ACCESS_CLIENT_ID")
	clientSecret := os.Getenv("CF_ACCESS_CLIENT_SECRET")

	if os.Getenv("DEBUG") != "" {
		configuration.Debug = true
	}

	if clientID == "" || clientSecret == "" {
		return nil, errors.Errorf("Cloudflare Access credentials are not set!")
	}

	// Use AddDefaultHeader to include the Cloudflare headers globally
	configuration.AddDefaultHeader("CF-Access-Client-Id", clientID)
	configuration.AddDefaultHeader("CF-Access-Client-Secret", clientSecret)

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
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return nil, "", &ClientError{StatusCode: statusCode, Message: "failed to list users from Authentik", innerError: err}
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
	statusCode := 500
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if err != nil {
		return nil, "", &ClientError{StatusCode: statusCode, Message: "failed to list groups from Authentik", innerError: err}
	}

	return paginatedGroups.Results, getNextCursorFromPagination(paginatedGroups.Pagination), nil
}

func (c *AuthentikClient) ListChildrenGroups(ctx *gin.Context, groupID string) (memberGroups []*authentik.Group, err error) {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	usedByModels, resp, err := c.client.CoreApi.CoreGroupsUsedByList(ctxWithAuth, groupID).Execute()
	if err != nil {
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return nil, &ClientError{StatusCode: statusCode, Message: "failed to get children groups for group from Authentik", innerError: err}
	}

	memberGroups = make([]*authentik.Group, 0)
	for _, usedByModel := range usedByModels {
		if usedByModel.ModelName == "group" {
			group, err := c.GetGroup(ctx, usedByModel.Pk)
			if err != nil {
				return nil, err
			}

			memberGroups = append(memberGroups, group)
		}
	}

	return memberGroups, nil
}

func (c *AuthentikClient) GetGroupUsers(ctx *gin.Context, groupID string) (members []authentik.GroupMember, err error) {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	group, resp, err := c.client.CoreApi.CoreGroupsRetrieve(ctxWithAuth, groupID).IncludeUsers(true).Execute()
	if err != nil {
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return nil, &ClientError{StatusCode: statusCode, Message: "failed to get users for group from Authentik", innerError: err}
	}

	return group.UsersObj, nil
}

func (c *AuthentikClient) GetGroup(ctx *gin.Context, groupID string) (group *authentik.Group, err error) {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	group, resp, err := c.client.CoreApi.CoreGroupsRetrieve(ctxWithAuth, groupID).IncludeUsers(false).Execute()
	if err != nil {
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return nil, &ClientError{StatusCode: statusCode, Message: "failed to get group from authentik", innerError: err}
	}

	return group, nil
}

func (c *AuthentikClient) AddUserToGroup(ctx *gin.Context, groupID string, userID string) error {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	// The user ID provided by Opal is the user's primary key in Authentik
	userPK, err := strconv.Atoi(userID)
	if err != nil {
		return err
	}
	userAccountRequest := authentik.NewUserAccountRequest(int32(userPK))

	resp, err := c.client.CoreApi.CoreGroupsAddUserCreate(ctxWithAuth, groupID).UserAccountRequest(*userAccountRequest).Execute()
	if err != nil {
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return &ClientError{StatusCode: statusCode, Message: "failed to add user to group in Authentik", innerError: err}
	}

	return err
}

func (c *AuthentikClient) RemoveUserFromGroup(ctx *gin.Context, groupID string, userID string) error {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)
	// The user ID provided by Opal is the user's primary key in Authentik
	userPK, err := strconv.Atoi(userID)
	if err != nil {
		return err
	}
	userAccountRequest := authentik.NewUserAccountRequest(int32(userPK))

	resp, err := c.client.CoreApi.CoreGroupsRemoveUserCreate(ctxWithAuth, groupID).UserAccountRequest(*userAccountRequest).Execute()
	if err != nil {
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return &ClientError{StatusCode: statusCode, Message: "failed to remove user from group in authentik", innerError: err}
	}

	return err
}

func (c *AuthentikClient) AddGroupToGroup(ctx *gin.Context, containingGroupID string, memberGroupID string) error {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)

	_, resp, err := c.client.CoreApi.CoreGroupsPartialUpdate(
		ctxWithAuth,
		memberGroupID,
	).PatchedGroupRequest(
		authentik.PatchedGroupRequest{
			Parent: *authentik.NewNullableString(&containingGroupID),
		},
	).Execute()
	if err != nil {
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return &ClientError{StatusCode: statusCode, Message: "Failed to add member group to containing group!", innerError: err}
	}

	return nil
}

func (c *AuthentikClient) RemoveGroupFromGroup(ctx *gin.Context, containingGroupID string, memberGroupID string) error {
	ctxWithAuth := c.addAuthTokenToCtx(ctx)

	_, resp, err := c.client.CoreApi.CoreGroupsPartialUpdate(
		ctxWithAuth,
		memberGroupID,
	).PatchedGroupRequest(
		authentik.PatchedGroupRequest{
			Parent: *authentik.NewNullableString(nil),
		},
	).Execute()
	if err != nil {
		statusCode := 500
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return &ClientError{StatusCode: statusCode, Message: "Failed to remove member group from containing group!", innerError: err}
	}

	return nil
}

func (c *AuthentikClient) addAuthTokenToCtx(ctx *gin.Context) context.Context {
	return context.WithValue(ctx, authentik.ContextAccessToken, c.token)
}

func getNextCursorFromPagination(pagination authentik.Pagination) string {
	// If on last page, return empty next cursor, which means all resources have been fetched
	if pagination.TotalPages == pagination.Current {
		return ""
	}

	return strconv.FormatFloat(float64(pagination.Next), 'f', 0, 32)
}

func getPageFromCtx(ctx *gin.Context) (int32, error) {
	page, err := strconv.Atoi(ctx.DefaultQuery(PageQueryParam, "1"))
	if err != nil {
		return -1, err
	}

	return int32(page), nil
}

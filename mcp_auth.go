package mcp

type Authorizer interface {
	Authorize(token string) (userID string, err error)
}

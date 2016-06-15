package config

import (
	"webup/backoops/options"

	"github.com/ncw/swift"
)

// GetSwiftConnection initialize a new Swift connection
func GetSwiftConnection(options options.SwiftOptions) (swift.Connection, error) {

	// Create a connection
	c := swift.Connection{
		UserName: options.User,
		ApiKey:   options.APIKey,
		AuthUrl:  options.AuthURL,
		Tenant:   options.TenantName, // Name of the tenant (v2 auth only)
	}
	// Authenticate
	err := c.Authenticate()

	return c, err
}

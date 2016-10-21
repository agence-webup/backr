package swift

import (
	"webup/backr"

	"github.com/ncw/swift"

	log "github.com/Sirupsen/logrus"
)

// GetSwiftConnection initialize a new Swift connection
func getSwiftConnection(opts backr.SwiftSettings) (*swift.Connection, error) {

	// Create a connection
	c := swift.Connection{
		UserName: opts.User,
		ApiKey:   opts.APIKey,
		AuthUrl:  opts.AuthURL,
		Tenant:   opts.TenantName, // Name of the tenant (v2 auth only)
	}
	// Authenticate
	err := c.Authenticate()

	log.Debugln("Connecting to Swift...")

	return &c, err
}

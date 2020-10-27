package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/mightyguava/dynamosql/schema"
)

func init() {
	sql.Register("dynamodb", &Driver{})
}

type Driver struct {
	cfg Config

	cfgOnce *sync.Once
	openErr error
}

var _ driver.Driver = &Driver{}
var _ driver.DriverContext = &Driver{}

type Config struct {
	Session *session.Session
}

func New(cfg Config) *Driver {
	return &Driver{
		cfg: cfg,
	}
}

func (d *Driver) Open(connStr string) (driver.Conn, error) {
	c, err := d.OpenConnector(connStr)
	if err != nil {
		return nil, err
	}
	return c.Connect(context.Background())
}

// OpenConnector initializes and returns a Connector. The db/sql package will call exactly once
// per sql.Open() call. Opening the connections to the database will use the returned Connector.
func (d *Driver) OpenConnector(connStr string) (driver.Connector, error) {
	var err error
	sess := d.cfg.Session
	if sess == nil {
		sess, err = session.NewSession(nil)
		if err != nil {
			return nil, err
		}
	}
	dynamo := dynamodb.New(sess)
	return &connector{
		dynamo: dynamo,
		driver: d,
		tables: schema.NewTableLoader(dynamo),
	}, nil
}

type connector struct {
	driver *Driver
	dynamo *dynamodb.DynamoDB
	tables *schema.TableLoader
}

var _ driver.Connector = &connector{}

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	return &conn{dynamo: c.dynamo, tables: c.tables}, nil
}

func (c *connector) Driver() driver.Driver {
	return c.driver
}

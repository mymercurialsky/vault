package radius

import (
	"strings"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathConfig(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: `config`,
		Fields: map[string]*framework.FieldSchema{
			"host": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "127.0.0.1",
				Description: "radius host to connect to (default: 127.0.0.1)",
			},

			"port": &framework.FieldSchema{
				Type:        framework.TypeInt,
				Default:     1812,
				Description: "radius port (default: 1812)",
			},
			"secret": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "secret shared with the radius server",
			},
			"enable_default_policy": &framework.FieldSchema{
				Type:        framework.TypeBool,
				Default:     true,
				Description: "enable granting default policy upon authentication (default: true)",
			},
			"dial_timeout": &framework.FieldSchema{
				Type:        framework.TypeDurationSecond,
				Default:     10,
				Description: "number of seconds before connect timeouts (default: 10)",
			},
			"read_timeout": &framework.FieldSchema{
				Type:        framework.TypeDurationSecond,
				Default:     10,
				Description: "number of seconds before response timeouts (default: 10)",
			},
			"nas_port": &framework.FieldSchema{
				Type:        framework.TypeInt,
				Default:     10,
				Description: "RADIUS NAS port field (default: 10)",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathConfigRead,
			logical.UpdateOperation: b.pathConfigWrite,
		},

		HelpSynopsis:    pathConfigHelpSyn,
		HelpDescription: pathConfigHelpDesc,
	}
}

/*
 * Construct ConfigEntry struct using stored configuration.
 */
func (b *backend) Config(req *logical.Request) (*ConfigEntry, error) {
	// Schema for ConfigEntry
	fd, err := b.getConfigFieldData()
	if err != nil {
		return nil, err
	}

	// Create a new ConfigEntry, filling in defaults where appropriate
	result, err := b.newConfigEntry(fd)
	if err != nil {
		return nil, err
	}

	storedConfig, err := req.Storage.Get("config")
	if err != nil {
		return nil, err
	}

	if storedConfig == nil {
		// No user overrides, return default configuration
		return result, nil
	}

	// Deserialize stored configuration.
	// Fields not specified in storedConfig will retain their defaults.
	if err := storedConfig.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *backend) pathConfigRead(
	req *logical.Request, d *framework.FieldData) (*logical.Response, error) {

	cfg, err := b.Config(req)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}

	resp := &logical.Response{
		Data: structs.New(cfg).Map(),
	}
	resp.AddWarning("Read access to this endpoint should be controlled via ACLs as it will return the configuration information as-is, including any passwords.")
	return resp, nil
}

/*
 * Creates and initializes a ConfigEntry object with its default values,
 * as specified by the passed schema.
 */
func (b *backend) newConfigEntry(d *framework.FieldData) (*ConfigEntry, error) {
	cfg := new(ConfigEntry)

	host := d.Get("host").(string)
	if host != "" {
		cfg.Host = strings.ToLower(host)
	}
	port := d.Get("port").(int)
	if port != 0 {
		cfg.Port = port
	}
	secret := d.Get("secret").(string)
	if secret != "" {
		cfg.Secret = secret
	}
	enable_default_policy := d.Get("enable_default_policy").(bool)
	if enable_default_policy {
		cfg.EnableDefaultPolicy = enable_default_policy
	}
	dial_timeout := d.Get("dial_timeout").(int)
	if dial_timeout != 0 {
		cfg.DialTimeout = dial_timeout
	}
	read_timeout := d.Get("read_timeout").(int)
	if read_timeout != 0 {
		cfg.ReadTimeout = read_timeout
	}
	nas_port := d.Get("nas_port").(int)
	if nas_port != 0 {
		cfg.NasPort = nas_port
	}

	return cfg, nil
}

func (b *backend) pathConfigWrite(
	req *logical.Request, d *framework.FieldData) (*logical.Response, error) {

	// Build a ConfigEntry struct out of the supplied FieldData
	cfg, err := b.newConfigEntry(d)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	entry, err := logical.StorageEntryJSON("config", cfg)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(entry); err != nil {
		return nil, err
	}

	return nil, nil
}

type ConfigEntry struct {
	Host                string `json:"host" structs:"host" mapstructure:"host"`
	Port                int    `json:"port" structs:"port" mapstructure:"port"`
	Secret              string `json:"secret" structs:"secret" mapstructure:"secret"`
	EnableDefaultPolicy bool   `json:"enable_default_policy" structs:"enable_default_policy" mapstructure:"enable_default_policy"`
	DialTimeout         int    `json:"dial_timeout" structs:"dial_timeout" mapstructure:"dial_timeout"`
	ReadTimeout         int    `json:"read_timeout" structs:"read_timeout" mapstructure:"read_timeout"`
	NasPort             int    `json:"nas_port" structs:"nas_port" mapstructure:"nas_port"`
}

/*
 * Returns FieldData describing our ConfigEntry struct schema
 */
func (b *backend) getConfigFieldData() (*framework.FieldData, error) {
	configPath := b.Route("config")

	if configPath == nil {
		return nil, logical.ErrUnsupportedPath
	}

	raw := make(map[string]interface{}, len(configPath.Fields))

	fd := framework.FieldData{
		Raw:    raw,
		Schema: configPath.Fields,
	}

	return &fd, nil
}

const pathConfigHelpSyn = `
Configure the RADIUS server to connect to, along with its options.
`

const pathConfigHelpDesc = `
This endpoint allows you to configure the RADIOS server to connect to and its
configuration options.

Upon successful authentication a users will be automatically granted 
the default policy unless disabled by enable_default_policy.
In that case, a user with no configured policies will be denied access
`

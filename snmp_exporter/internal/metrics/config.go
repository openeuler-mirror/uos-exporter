package metrics

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"gopkg.in/yaml.v2"
)

func LoadFile(paths []string, expandEnvVars bool) (*Config, error) {
	cfg := &Config{}
	if err := loadConfigFiles(paths, cfg); err != nil {
		return nil, err
	}

	if expandEnvVars {
		if err := expandAuthEnvVars(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func loadConfigFiles(paths []string, cfg *Config) error {
	for _, p := range paths {
		files, err := filepath.Glob(p)
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := loadSingleConfigFile(f, cfg); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadSingleConfigFile(filePath string, cfg *Config) error {
	cleanPath := filepath.Clean(filePath)
	if !strings.HasSuffix(cleanPath, "yaml") && !strings.HasSuffix(cleanPath, "yml") {
		return fmt.Errorf("invalid config path: %s", filePath)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.UnmarshalStrict(content, cfg)
}

func expandAuthEnvVars(cfg *Config) error {
	for i, auth := range cfg.Auths {
		if err := expandSingleAuthEnvVars(cfg.Auths[i], *auth); err != nil {
			return err
		}
	}
	return nil
}

func expandSingleAuthEnvVars(auth *Auth, original Auth) error {
	var err error

	if original.Username != "" {
		auth.Username, err = substituteEnvVariables(original.Username)
		if err != nil {
			return err
		}
	}

	if original.Password != "" {
		password, err := substituteEnvVariables(string(original.Password))
		if err != nil {
			return err
		}
		auth.Password.Set(password)
	}

	if original.PrivPassword != "" {
		privPassword, err := substituteEnvVariables(string(original.PrivPassword))
		if err != nil {
			return err
		}
		auth.PrivPassword.Set(privPassword)
	}

	return nil
}

var (
	defaultRetries = 3

	DefaultAuth = Auth{
		Community:     "public",
		SecurityLevel: "noAuthNoPriv",
		AuthProtocol:  "MD5",
		PrivProtocol:  "DES",
		Version:       2,
	}
	DefaultWalkParams = WalkParams{
		MaxRepetitions:          25,
		Retries:                 &defaultRetries,
		Timeout:                 time.Second * 5,
		UseUnconnectedUDPSocket: false,
		AllowNonIncreasingOIDs:  false,
	}
	DefaultModule = Module{
		WalkParams: DefaultWalkParams,
	}
	DefaultRegexpExtract = RegexpExtract{
		Value: "$1",
	}
)

// Config for the snmp_exporter.
type Config struct {
	Auths   map[string]*Auth   `yaml:"auths,omitempty"`
	Modules map[string]*Module `yaml:"modules,omitempty"`
	Version int                `yaml:"version,omitempty"`
}

type WalkParams struct {
	MaxRepetitions          uint32        `yaml:"max_repetitions,omitempty"`
	Retries                 *int          `yaml:"retries,omitempty"`
	Timeout                 time.Duration `yaml:"timeout,omitempty"`
	UseUnconnectedUDPSocket bool          `yaml:"use_unconnected_udp_socket,omitempty"`
	AllowNonIncreasingOIDs  bool          `yaml:"allow_nonincreasing_oids,omitempty"`
}

type Module struct {
	// A list of OIDs.
	Walk       []string        `yaml:"walk,omitempty"`
	Get        []string        `yaml:"get,omitempty"`
	Metrics    []*Metric       `yaml:"metrics"`
	WalkParams WalkParams      `yaml:",inline"`
	Filters    []DynamicFilter `yaml:"filters,omitempty"`
}

func (c *Module) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultModule
	type plain Module
	return unmarshal((*plain)(c))
}

// ConfigureSNMP sets the various version and auth settings.
func (c Auth) ConfigureSNMP(g *gosnmp.GoSNMP, snmpContext string) {
	c.setSNMPVersion(g)
	c.setCommunityAndContext(g, snmpContext)
	c.configureSecurityParameters(g)
}

func (c Auth) setSNMPVersion(g *gosnmp.GoSNMP) {
	switch c.Version {
	case 1:
		g.Version = gosnmp.Version1
	case 2:
		g.Version = gosnmp.Version2c
	case 3:
		g.Version = gosnmp.Version3
	}
}

func (c Auth) setCommunityAndContext(g *gosnmp.GoSNMP, snmpContext string) {
	g.Community = string(c.Community)
	if snmpContext == "" {
		g.ContextName = c.ContextName
	} else {
		g.ContextName = snmpContext
	}
}

func (c Auth) configureSecurityParameters(g *gosnmp.GoSNMP) {
	g.SecurityModel = gosnmp.UserSecurityModel
	usm := &gosnmp.UsmSecurityParameters{UserName: c.Username}

	auth, priv := c.determineSecurityLevel(g)
	if auth {
		c.configureAuthProtocol(usm)
	}
	if priv {
		c.configurePrivProtocol(usm)
	}

	g.SecurityParameters = usm
}

func (c Auth) determineSecurityLevel(g *gosnmp.GoSNMP) (auth, priv bool) {
	switch c.SecurityLevel {
	case "noAuthNoPriv":
		g.MsgFlags = gosnmp.NoAuthNoPriv
	case "authNoPriv":
		g.MsgFlags = gosnmp.AuthNoPriv
		auth = true
	case "authPriv":
		g.MsgFlags = gosnmp.AuthPriv
		auth, priv = true, true
	}
	return
}

func (c Auth) configureAuthProtocol(usm *gosnmp.UsmSecurityParameters) {
	usm.AuthenticationPassphrase = string(c.Password)
	switch c.AuthProtocol {
	case "SHA":
		usm.AuthenticationProtocol = gosnmp.SHA
	case "SHA224":
		usm.AuthenticationProtocol = gosnmp.SHA224
	case "SHA256":
		usm.AuthenticationProtocol = gosnmp.SHA256
	case "SHA384":
		usm.AuthenticationProtocol = gosnmp.SHA384
	case "SHA512":
		usm.AuthenticationProtocol = gosnmp.SHA512
	case "MD5":
		usm.AuthenticationProtocol = gosnmp.MD5
	}
}

func (c Auth) configurePrivProtocol(usm *gosnmp.UsmSecurityParameters) {
	usm.PrivacyPassphrase = string(c.PrivPassword)
	switch c.PrivProtocol {
	case "DES":
		usm.PrivacyProtocol = gosnmp.DES
	case "AES":
		usm.PrivacyProtocol = gosnmp.AES
	case "AES192":
		usm.PrivacyProtocol = gosnmp.AES192
	case "AES192C":
		usm.PrivacyProtocol = gosnmp.AES192C
	case "AES256":
		usm.PrivacyProtocol = gosnmp.AES256
	case "AES256C":
		usm.PrivacyProtocol = gosnmp.AES256C
	}
}

type Filters struct {
	Static  []StaticFilter  `yaml:"static,omitempty"`
	Dynamic []DynamicFilter `yaml:"dynamic,omitempty"`
}

type StaticFilter struct {
	Targets []string `yaml:"targets,omitempty"`
	Indices []string `yaml:"indices,omitempty"`
}
type DynamicFilter struct {
	Oid     string   `yaml:"oid"`
	Targets []string `yaml:"targets,omitempty"`
	Values  []string `yaml:"values,omitempty"`
}

type Metric struct {
	Name            string                     `yaml:"name"`
	Oid             string                     `yaml:"oid"`
	Type            string                     `yaml:"type"`
	Help            string                     `yaml:"help"`
	Indexes         []*Index                   `yaml:"indexes,omitempty"`
	Lookups         []*Lookup                  `yaml:"lookups,omitempty"`
	RegexpExtracts  map[string][]RegexpExtract `yaml:"regex_extracts,omitempty"`
	DateTimePattern string                     `yaml:"datetime_pattern,omitempty"`
	EnumValues      map[int]string             `yaml:"enum_values,omitempty"`
	Offset          float64                    `yaml:"offset,omitempty"`
	Scale           float64                    `yaml:"scale,omitempty"`
}

type Index struct {
	Labelname  string         `yaml:"labelname"`
	Type       string         `yaml:"type"`
	FixedSize  int            `yaml:"fixed_size,omitempty"`
	Implied    bool           `yaml:"implied,omitempty"`
	EnumValues map[int]string `yaml:"enum_values,omitempty"`
}

type Lookup struct {
	Labels    []string `yaml:"labels"`
	Labelname string   `yaml:"labelname"`
	Oid       string   `yaml:"oid,omitempty"`
	Type      string   `yaml:"type,omitempty"`
}

// Secret is a string that must not be revealed on marshaling.
type Secret string

func (s *Secret) Set(value string) {
	*s = Secret(value)
}

// Hack for creating snmp.yml with the secret.
var (
	DoNotHideSecrets = false
)

// MarshalYAML implements the yaml.Marshaler interface.
func (s Secret) MarshalYAML() (interface{}, error) {
	if DoNotHideSecrets {
		return string(s), nil
	}
	if s != "" {
		return "<secret>", nil
	}
	return nil, nil
}

type Auth struct {
	Community     Secret `yaml:"community,omitempty"`
	SecurityLevel string `yaml:"security_level,omitempty"`
	Username      string `yaml:"username,omitempty"`
	Password      Secret `yaml:"password,omitempty"`
	AuthProtocol  string `yaml:"auth_protocol,omitempty"`
	PrivProtocol  string `yaml:"priv_protocol,omitempty"`
	PrivPassword  Secret `yaml:"priv_password,omitempty"`
	ContextName   string `yaml:"context_name,omitempty"`
	Version       int    `yaml:"version,omitempty"`
}

func (c *Auth) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultAuth
	type plain Auth
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if err := c.validateVersion(); err != nil {
		return err
	}

	if c.Version == 3 {
		return c.validateV3Security()
	}

	return nil
}

func (c *Auth) validateVersion() error {
	if c.Version < 1 || c.Version > 3 {
		return fmt.Errorf("SNMP version must be 1, 2 or 3. Got: %d", c.Version)
	}
	return nil
}

func (c *Auth) validateV3Security() error {
	switch c.SecurityLevel {
	case "authPriv":
		if err := c.validatePrivParams(); err != nil {
			return err
		}
		fallthrough
	case "authNoPriv":
		if err := c.validateAuthParams(); err != nil {
			return err
		}
		fallthrough
	case "noAuthNoPriv":
		if c.Username == "" {
			return fmt.Errorf("auth username is missing, required for SNMPv3")
		}
	default:
		return fmt.Errorf("security level must be one of authPriv, authNoPriv or noAuthNoPriv")
	}
	return nil
}

func (c *Auth) validatePrivParams() error {
	if c.PrivPassword == "" {
		return fmt.Errorf("priv password is missing, required for SNMPv3 with priv")
	}
	if !isValidPrivProtocol(c.PrivProtocol) {
		return fmt.Errorf("priv protocol must be DES or AES")
	}
	return nil
}

func (c *Auth) validateAuthParams() error {
	if c.Password == "" {
		return fmt.Errorf("auth password is missing, required for SNMPv3 with auth")
	}
	if !isValidAuthProtocol(c.AuthProtocol) {
		return fmt.Errorf("auth protocol must be SHA or MD5")
	}
	return nil
}

func isValidPrivProtocol(protocol string) bool {
	switch protocol {
	case "DES", "AES", "AES192", "AES192C", "AES256", "AES256C":
		return true
	}
	return false
}

func isValidAuthProtocol(protocol string) bool {
	switch protocol {
	case "MD5", "SHA", "SHA224", "SHA256", "SHA384", "SHA512":
		return true
	}
	return false
}

type RegexpExtract struct {
	Value string `yaml:"value"`
	Regex Regexp `yaml:"regex"`
}

func (c *RegexpExtract) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultRegexpExtract
	type plain RegexpExtract
	return unmarshal((*plain)(c))
}

// Regexp encapsulates a regexp.Regexp and makes it YAML marshalable.
type Regexp struct {
	*regexp.Regexp
}

// MarshalYAML implements the yaml.Marshaler interface.
func (re Regexp) MarshalYAML() (interface{}, error) {
	if re.Regexp != nil {
		return re.String(), nil
	}
	return nil, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (re *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	regex, err := regexp.Compile("^(?:" + s + ")$")
	if err != nil {
		return err
	}
	re.Regexp = regex
	return nil
}

func substituteEnvVariables(value string) (string, error) {
	result := os.Expand(value, func(s string) string {
		return os.Getenv(s)
	})
	if result == "" {
		return "", errors.New(value + " environment variable not found")
	}
	return result, nil
}

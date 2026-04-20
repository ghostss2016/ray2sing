package ray2sing

import (
	"net/url"
	"regexp"
	"strings"

	T "github.com/sagernet/sing-box/option"
)

// HysteriaURLData holds the parsed data from a Hysteria URL.
type UrlSchema struct {
	Scheme      string
	Username    string
	Password    string
	Hostname    string
	Port        uint16
	ServerPorts []string // sing-box format: "20000:40000" or ["443","20000:40000"]
	Name        string
	Params      map[string]string
}

func (u UrlSchema) GetServerOption() T.ServerOptions {
	return T.ServerOptions{
		Server:     u.Hostname,
		ServerPort: u.Port,
	}
}

// func (u UrlSchema) GetRelayOptions() (*T.TurnRelayOptions, error) {
// 	return ParseTurnURL(u.Params["relay"])
// }

// parseHysteria2 parses a given URL and returns a HysteriaURLData struct.
// Supports Hysteria2 port hopping via ":port1,port2-port3" syntax in URL host.
func ParseUrl(inputURL string, defaultPort uint16) (*UrlSchema, error) {
	// Pre-process: extract port-range (e.g. "20000-40000" or "443,20000-30000")
	// from raw URL because url.Parse can't handle non-integer port.
	inputURL, rawPortField := extractPortRange(inputURL, defaultPort)

	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return nil, err
	}
	port := toUInt16(parsedURL.Port(), defaultPort)
	serverPorts := convertPortRangeToSingbox(rawPortField)

	data := &UrlSchema{
		Scheme:      parsedURL.Scheme,
		Username:    parsedURL.User.Username(),
		Password:    getPassword(parsedURL),
		Hostname:    parsedURL.Hostname(),
		Port:        port,
		ServerPorts: serverPorts,
		Name:        parsedURL.Fragment,
		Params:      make(map[string]string),
	}
	if isBase64CharsOnly(data.Username) {
		userInfo, err := decodeBase64IfNeeded(data.Username)

		// fmt.Print(userInfo)
		if err == nil && isValidChar(userInfo) {
			// If decoding is successful, use the decoded string
			userDetails := strings.Split(userInfo, ":")
			if len(userDetails) == 2 {
				data.Username = userDetails[0]
				data.Password = userDetails[1]
			}
		}
	}

	for key, values := range parsedURL.Query() {
		data.Params[strings.ReplaceAll(strings.ToLower(key), "_", "")] = strings.Join(values, ",")
	}

	return data, nil
}

func getPassword(u *url.URL) string {
	if password, ok := u.User.Password(); ok {
		return password
	}
	return ""
}

var base64CharRegex = regexp.MustCompile(`^[A-Za-z0-9+/=]+$`)

func isBase64CharsOnly(s string) bool {
	return base64CharRegex.MatchString(s)
}

var validCharRegex = regexp.MustCompile(`^[A-Za-z0-9+/=_)(: !~@#$%^&*-]+$`)

func isValidChar(s string) bool {

	return validCharRegex.MatchString(s)
}

// portRangeRe matches "host:portExpr" where portExpr contains digits, commas or hyphens.
// e.g. "example.com:20000-40000" or "example.com:443,20000-30000"
var portRangeRe = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+\-.]*://(?:[^@/?#]*@)?(?:\[[^\]]+\]|[^:/?#\s]+)):(\d{1,5}(?:[-,]\d{1,5})+)`)

// extractPortRange detects comma/hyphen port list in URL and replaces with
// a single numeric port (first valid one found) so net/url can parse it.
// Returns modified URL + original raw port field (empty if no range).
func extractPortRange(rawURL string, defaultPort uint16) (string, string) {
	m := portRangeRe.FindStringSubmatchIndex(rawURL)
	if m == nil {
		return rawURL, ""
	}
	// m[2]:m[3] = host-prefix, m[4]:m[5] = port field
	portField := rawURL[m[4]:m[5]]
	// Find first numeric port for url.Parse to succeed
	firstPort := ""
	for _, chunk := range strings.FieldsFunc(portField, func(r rune) bool { return r == ',' || r == '-' }) {
		if chunk != "" {
			firstPort = chunk
			break
		}
	}
	if firstPort == "" {
		firstPort = "443"
	}
	// Replace port field in URL with first port
	modified := rawURL[:m[4]] + firstPort + rawURL[m[5]:]
	return modified, portField
}

// convertPortRangeToSingbox converts "20000-40000" -> ["20000:40000"],
// "443,20000-30000" -> ["443","20000:30000"]
func convertPortRangeToSingbox(portField string) []string {
	if portField == "" {
		return nil
	}
	var out []string
	for _, item := range strings.Split(portField, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		// Replace hyphen with colon (sing-box range format)
		out = append(out, strings.ReplaceAll(item, "-", ":"))
	}
	return out
}

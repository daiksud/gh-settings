package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Environment list pages may include 100 full objects, including reviewer
// profiles. Bound each page while leaving room for that larger response shape.
const environmentResponseLimit = 16 << 20

// JSON decoding must not turn duplicate members, invalid UTF-8, or unpaired
// UTF-16 escapes into an apparently complete observation or acknowledgment.
func environmentJSON(data []byte, out any) error {
	if !validEnvironmentJSONUnicode(data) || webhookJSON(data, out) != nil {
		return fmt.Errorf("invalid or ambiguous Environment response")
	}
	return nil
}

// Some inventory or principal objects carry unrelated metadata. Keep those
// fields unowned, but reject aliases for the exact keys used as evidence.
func rejectEnvironmentAliases(fields map[string]json.RawMessage, canonical ...string) error {
	for name := range fields {
		for _, key := range canonical {
			if name != key && strings.EqualFold(name, key) {
				return fmt.Errorf("Environment response contains an ambiguous field")
			}
		}
	}
	return nil
}

func validEnvironmentJSONUnicode(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	inString := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[i] != '\\' {
			continue
		}
		if i+1 >= len(data) {
			return false
		}
		i++
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			return false
		}
		value, err := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if value >= 0xdc00 && value <= 0xdfff {
			return false
		}
		if value >= 0xd800 && value <= 0xdbff {
			if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}

func (c *EnvironmentsClient) envRequestStrict(method, path string, body io.Reader) (map[string]json.RawMessage, error) {
	fields, _, err := c.envRequestStrictWithLinks(method, path, body)
	return fields, err
}

func (c *EnvironmentsClient) envRequestStrictWithLinks(method, path string, body io.Reader) (map[string]json.RawMessage, []string, error) {
	response, err := c.rest.Request(method, path, body)
	if err != nil {
		return nil, nil, fmt.Errorf("Environment request unavailable")
	}
	defer response.Body.Close()
	// Environment detail/list, parent PUT and pattern POST each document 200.
	// Do not treat another successful HTTP status as a complete observation.
	if response.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected Environment response status %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, environmentResponseLimit+1))
	var fields map[string]json.RawMessage
	if err != nil || len(data) > environmentResponseLimit || environmentJSON(data, &fields) != nil || fields == nil {
		return nil, nil, fmt.Errorf("Environment response incomplete or unavailable")
	}
	return fields, response.Header.Values("Link"), nil
}

func (c *EnvironmentsClient) envReadStrict(path string) (map[string]json.RawMessage, error) {
	return c.envRequestStrict(http.MethodGet, path, nil)
}

// Link URLs are evidence only and are never followed. An absent header is
// compatible with count-based paging; a present header must agree with it.
func (c *EnvironmentsClient) envReadPage(path string) (map[string]json.RawMessage, bool, bool, error) {
	fields, links, err := c.envRequestStrictWithLinks(http.MethodGet, path, nil)
	if err != nil {
		return nil, false, false, err
	}
	for _, link := range links {
		if strings.TrimSpace(link) == "" {
			return nil, false, false, fmt.Errorf("invalid Environment pagination metadata")
		}
	}
	next, err := webhookHasNext(strings.Join(links, ","))
	if err != nil {
		return nil, false, false, fmt.Errorf("invalid Environment pagination metadata")
	}
	return fields, next, len(links) != 0, nil
}

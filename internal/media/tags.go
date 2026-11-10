package media

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// TagValue accepts string and number values, both of which ffprobe emits for tags
type TagValue string

func (t *TagValue) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = TagValue(s)
		return nil
	}

	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*t = TagValue(n.String())
	return nil
}

func (t TagValue) String() string { return string(t) }

type tagMap map[string]TagValue

func (m tagMap) toStrings() map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v.String()
	}
	return out
}

// lookup finds a tag case-insensitively: tag casing depends on the container
func (m tagMap) lookup(key string) (string, bool) {
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v.String(), true
		}
	}
	return "", false
}

func parseSeconds(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

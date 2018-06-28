package websoketkit_test

import (
	"encoding/json"
	"testing"

	"github.com/tidwall/gjson"
)

func Test_func(t *testing.T) {
	html := `{"action": "fun","funcname": "test","parame": {"x":"y"}}`
	datajson := gjson.Get(html, "parame").String()
	v := make(map[string]interface{}, 0)
	err := json.Unmarshal([]byte(datajson), &v)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(v)
	}
}

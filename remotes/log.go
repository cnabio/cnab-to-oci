package remotes

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

func logPayload(payload interface{}) {
	buf, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	logrus.Info(string(buf))
}

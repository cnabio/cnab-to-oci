package remotes

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONMessage is a simple JSON message structure for displaying progress
type JSONMessage struct {
	Status         string `json:"status,omitempty"`
	Progress       string `json:"progress,omitempty"`
	ProgressDetail struct {
		Current int64 `json:"current,omitempty"`
		Total   int64 `json:"total,omitempty"`
	} `json:"progressDetail,omitempty"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// DisplayJSONMessagesStream reads JSON messages from the input stream and writes them to output.
// This is a simplified version of docker/docker/pkg/jsonmessage.DisplayJSONMessagesStream
func DisplayJSONMessagesStream(in io.ReadCloser, out io.Writer) error {
	decoder := json.NewDecoder(in)
	for {
		var msg JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if msg.Error != "" {
			return fmt.Errorf("%s", msg.Error)
		}

		// Display the message
		if msg.Status != "" {
			if msg.ID != "" {
				fmt.Fprintf(out, "%s: %s", msg.ID, msg.Status)
			} else {
				fmt.Fprint(out, msg.Status)
			}
			if msg.Progress != "" {
				fmt.Fprintf(out, " %s", msg.Progress)
			}
			fmt.Fprintln(out)
		}
	}
	return nil
}

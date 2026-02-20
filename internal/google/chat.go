package google

import "fmt"

// SendDM sends a direct message to the specified user via Google Chat.
func SendDM(email, message string) error {
	_, err := Run("chat", "dm", "send", email, "--text", message)
	if err != nil {
		return fmt.Errorf("send dm to %s: %w", email, err)
	}
	return nil
}

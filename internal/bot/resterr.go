package bot

import (
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// wrapRESTErr augments a Discord REST error with an actionable hint for the
// most common authentication failures. When err is a *discordgo.RESTError
// with a 401 (invalid/expired bot token) or 403 (missing permissions) status,
// it returns a detailed message explaining the cause and how to fix it. All
// other errors — and nil — are returned wrapped with msg as usual, so the
// original Discord error is never lost.
func wrapRESTErr(msg string, err error) error {
	if err == nil {
		return nil
	}

	restErr, ok := err.(*discordgo.RESTError)
	if !ok {
		return fmt.Errorf("%s: %v", msg, err)
	}

	switch restErr.Response.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf(
			"%s: unauthorized (HTTP 401). The bot token is likely invalid, expired, or was rotated. Regenerate it in the Developer Portal (Bot -> Token -> Reset Token) and update it in your config. Original error: %v",
			msg, err,
		)
	case http.StatusForbidden:
		return fmt.Errorf(
			"%s: forbidden (HTTP 403). The bot lacks the required permission, or the client_id is incorrect. Ensure the app has the \"applications.commands\" scope and the bot has the right permissions in the Developer Portal. Original error: %v",
			msg, err,
		)
	default:
		return fmt.Errorf("%s: %v", msg, err)
	}
}

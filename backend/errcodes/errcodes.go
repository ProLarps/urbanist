// Package errcodes defines the numeric error vocabulary shared by the HTTP
// API and the WebSocket layer. Error bodies carry the numeric code instead
// of a string, so clients match on stable integers (see docs/API.md §2.6).
package errcodes

// Code is a stable numeric error identifier; never renumber released codes.
type Code int

const (
	OK                   Code = iota
	BadRequest                // malformed body, bad name, bad parameters
	Unauthorized              // token missing, malformed or expired
	Forbidden                 // the action belongs to another player/room owner
	NotFound                  // generic unknown id
	RoomNotFound              // the referenced room code does not exist
	RoomFull                  // room capacity is exhausted
	RoomBusy                  // wrong room mode: joining or restarting a run room
	NotMember                 // acting on a room the player is not in
	NotEveryoneConnected      // start pressed while a member has no live socket
	Internal

	total // must be last
)

// httpStatus maps codes onto wire status; keep it total over all codes.
var httpStatus = [total]int{
	OK:                   200,
	BadRequest:           400,
	Unauthorized:         401,
	Forbidden:            403,
	NotFound:             404,
	RoomNotFound:         404,
	RoomFull:             409,
	RoomBusy:             409,
	NotMember:            403,
	NotEveryoneConnected: 409,
	Internal:             500,
}

// messages exist purely for logs and human eyes; the wire format is numeric.
var messages = [total]string{
	OK:                   "ok",
	BadRequest:           "bad request",
	Unauthorized:         "unauthorized",
	Forbidden:            "forbidden",
	NotFound:             "not found",
	RoomNotFound:         "room not found",
	RoomFull:             "room is full",
	RoomBusy:             "room is in the wrong mode for that",
	NotMember:            "you are not a member of that room",
	NotEveryoneConnected: "not everyone is connected yet",
	Internal:             "internal error",
}

// HTTP is the HTTP status an error response should use.
func (c Code) HTTP() int {
	if c <= 0 || c >= total {
		return httpStatus[Internal]
	}
	return httpStatus[c]
}

// Message returns the human-readable text for logs and toasts.
func (c Code) Message() string {
	if c <= 0 || c >= total {
		return messages[Internal]
	}
	return messages[c]
}

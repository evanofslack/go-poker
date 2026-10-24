package table

// Error is a command refused. It goes only to whoever sent it.
type Error struct {
	Code    string `json:"code,omitempty"` // for errors the client handles
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }

// ChatMessage is a chat line. Name is "system" for the table's own.
type ChatMessage struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Time    int64  `json:"time"` // unix ms
}

// SystemName is who the table's own chat messages are from.
const SystemName = "system"

// NewToken is the secret that reclaims a person's seat, or empty once they
// gave it up.
type NewToken struct {
	Token string `json:"token"`
}

// NewPassword is the table's password after the host changed it, for
// everyone at the table to share in invite links. Empty when removed.
type NewPassword struct {
	Password string `json:"password"`
}

// HandsPage is a page of hand history.
type HandsPage struct {
	Hands []HandRecord `json:"hands"`
	More  bool         `json:"more"` // older hands are kept
}

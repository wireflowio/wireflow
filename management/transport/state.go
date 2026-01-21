package transport

// ConnectionState is an enum showing the state of a ICE Connection
type ConnectionState int

// List of supported States
const (
	ConnectionStateUnknown ConnectionState = iota

	// ConnectionStateNew probe the remote peer.
	ConnectionStateNew

	ConnectionStateChecking

	ConnectionStateConnected

	ConnectionStateCompleted

	ConnectionStateFailed

	ConnectionStateDisconnected

	ConnectionStateClosed
)

func (c ConnectionState) String() string {
	switch c {
	case ConnectionStateNew:
		return "New"
	case ConnectionStateChecking:
		return "Checking"
	case ConnectionStateConnected:
		return "Connected"
	case ConnectionStateCompleted:
		return "Completed"
	case ConnectionStateFailed:
		return "Failed"
	case ConnectionStateDisconnected:
		return "Disconnected"
	case ConnectionStateClosed:
		return "Closed"
	default:
		return "Invalid"
	}
}

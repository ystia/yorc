package hostspool

//go:generate go-enum -f=hostspool_structs.go --lower

import (
	"bytes"
	"encoding/json"
)

// HostStatus x ENUM(
// Free,
// Allocated
// )
type HostStatus int

// MarshalJSON is used to represent this enumeration as a string instead of an int
func (hs HostStatus) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(hs.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON is used to read this enumeration from a string
func (hs *HostStatus) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*hs, err = ParseHostStatus(s)
	return err
}

// TODO support winrm for windows hosts

// A Connection holds info used to connect to an host using SSH
type Connection struct {
	// The User that we should use for the connection. Defaults to root.
	User string `json:"user,omitempty"`
	// The Password that we should use for the connection. One of Password or PrivateKey is required. PrivateKey takes the precedence.
	Password string `json:"password,omitempty"`
	// The SSH Private Key that we should use for the connection. One of Password or PrivateKey is required. PrivateKey takes the precedence.
	PrivateKey string `json:"private_key,omitempty"`
	// The address of the Host to connect to. Defaults to the hostname specified during the registration.
	Host string `json:"host,omitempty"`
	// The Port to connect to. Defaults to 22 if set to 0.
	Port uint64 `json:"port,omitempty"`
}

// An Host holds information on an Host as it is known by the hostspool
type Host struct {
	Name       string            `json:"name,omitempty"`
	Connection Connection        `json:"connection"`
	Status     HostStatus        `json:"status"`
	Tags       map[string]string `json:"tags,omitempty"`
}

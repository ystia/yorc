package events

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"time"
)

//go:generate stringer -type=FieldType,LogLevel -output=log_entries_string.go

// FormattedLogEntry is the log entry representation
type FormattedLogEntry struct {
	level          LogLevel
	deploymentID   string
	additionalInfo OptionalFields
	content        []byte
}

// OptionalFields are log's additional info
type OptionalFields map[FieldType]interface{}

// FieldType is allowed/expected additional info types
type FieldType int

const (
	// WorkFlowID is the field type representing the workflow ID in formatted log entry
	WorkFlowID FieldType = iota

	// ExecutionID is the field type representing the execution ID in formatted log entry
	ExecutionID

	// NodeID is the field type representing the node ID in formatted log entry
	NodeID

	// InstanceID is the field type representing the instance ID in formatted log entry
	InstanceID

	// InterfaceID is the field type representing the interface ID in formatted log entry
	InterfaceID

	// OperationID is the field type representing the operation ID in formatted log entry
	OperationID

	// TypeID is the field type representing the type ID in formatted log entry
	TypeID
)

const storageMaxAllowedSize int = 512 * 1000

// LogLevel represents the log level enumeration
type LogLevel int

const (
	// INFO is the informative log level
	INFO LogLevel = iota

	// DEBUG is the debugging log level
	DEBUG

	// TRACE is the tracing log level
	TRACE

	// WARN is the warning log level
	WARN

	// ERROR is the error log level
	ERROR
)

// NewLogEntry allows to return a FormattedLogEntry instance with log level and deploymentID
func NewLogEntry(level LogLevel, deploymentID string) (*FormattedLogEntry, error) {
	if deploymentID == "" {
		return nil, errors.New("the deploymentID parameter must be filled")
	}

	return &FormattedLogEntry{
		level:        level,
		deploymentID: deploymentID,
	}, nil
}

// WithOptionalFields allows to return a FormattedLogEntry instance with additional fields
func WithOptionalFields(fields OptionalFields) *FormattedLogEntry {
	info := make(OptionalFields, len(fields))
	e := &FormattedLogEntry{additionalInfo: info}
	for k, v := range fields {
		info[k] = v
	}

	return e
}

// Add allows to add main fields to a formatted log entry
func (e FormattedLogEntry) Add(level LogLevel, deploymentID string) *FormattedLogEntry {
	e.level = level
	e.deploymentID = deploymentID
	return &e
}

// Register allows to register a formatted log entry with content
func (e FormattedLogEntry) Register(content []byte) error {
	if len(content) == 0 {
		return errors.New("the content parameter must be filled")
	}
	if e.deploymentID == "" {
		return errors.New("the deploymentID parameter must be filled")
	}
	e.content = content
	err := consulutil.StoreConsulKey(e.generateKey(), e.generateValue())
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to register log in consul for entry [%+v]", e))
	}
	return nil
}

// RunBufferedRegistration allows to run a registration with a buffered writer
func (e FormattedLogEntry) RunBufferedRegistration(buf BufferedLogEntryWriter, quit chan bool) error {
	if e.deploymentID == "" {
		return errors.New("the deploymentID parameter must be filled")
	}

	buf.run(quit, e)
	return nil
}

func (e FormattedLogEntry) generateKey() string {
	return path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "logs", time.Now().Format(time.RFC3339Nano))
}

func (e FormattedLogEntry) generateValue() []byte {
	// For presentation purpose, the formatted log entry is cast to flat map
	b, err := json.Marshal(e.toFlatMap())
	if err != nil {
		log.Printf("Failed to marshal entry [%+v]: due to error:%+v", e, err)
	}
	if len(b) > storageMaxAllowedSize {
		log.Printf("the max allowed size has been reached: truncation will be done to %d bytes", storageMaxAllowedSize)
		return b[:storageMaxAllowedSize]
	}
	return b
}

func (e FormattedLogEntry) toFlatMap() map[string]interface{} {
	flatMap := make(map[string]interface{})

	// Add main attributes from FormattedLogEntry
	flatMap["deploymentID"] = e.deploymentID
	flatMap["level"] = e.level.String()
	flatMap["content"] = string(e.content)

	// Add additional info
	for k, v := range e.additionalInfo {
		flatMap[k.String()] = v
	}
	return flatMap
}

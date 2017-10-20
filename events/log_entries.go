package events

import (
	"encoding/json"
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
	additionalInfo LogOptionalFields
	content        []byte
}

// FormattedLogEntryDraft is a partial FormattedLogEntry with only optional fields.
// It has to be completed with level and deploymentID
type FormattedLogEntryDraft struct {
	additionalInfo LogOptionalFields
}

// LogOptionalFields are log's additional info
type LogOptionalFields map[FieldType]interface{}

// FieldType is allowed/expected additional info types
type FieldType int

// This is used to retrieve timestamp
var getTimestamp = func() string {
	return time.Now().Format(time.RFC3339)
}

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

// Max allowed storage size in Consul kv for value is Equal to 512 Kb
// We approximate all data except the content value to be equal to 1Kb
const contentMaxAllowedValueSize int = 511 * 1000

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

// SimpleLogEntry allows to return a FormattedLogEntry instance with log level and deploymentID
func SimpleLogEntry(level LogLevel, deploymentID string) *FormattedLogEntry {
	return &FormattedLogEntry{
		level:        level,
		deploymentID: deploymentID,
	}
}

// WithOptionalFields allows to return a FormattedLogEntry instance with additional fields
func WithOptionalFields(fields LogOptionalFields) *FormattedLogEntryDraft {
	info := make(LogOptionalFields, len(fields))
	fle := &FormattedLogEntryDraft{additionalInfo: info}
	for k, v := range fields {
		info[k] = v
	}

	return fle
}

// NewLogEntry allows to add main fields to a formatted log entry
func (e FormattedLogEntryDraft) NewLogEntry(level LogLevel, deploymentID string) *FormattedLogEntry {
	return &FormattedLogEntry{
		level:          level,
		deploymentID:   deploymentID,
		additionalInfo: e.additionalInfo,
	}
}

// Register allows to register a formatted log entry with byte array content
func (e FormattedLogEntry) Register(content []byte) {
	if len(content) == 0 {
		log.Panic("The content parameter must be filled")
	}
	if e.deploymentID == "" {
		log.Panic("The deploymentID parameter must be filled")
	}
	e.content = content
	err := consulutil.StoreConsulKey(e.generateKey(), e.generateValue())
	if err != nil {
		log.Printf("Failed to register log in consul for entry:%+v due to error:%+v", e, err)
	}
}

// RegisterAsString allows to register a formatted log entry with string content
func (e FormattedLogEntry) RegisterAsString(content string) {
	e.Register([]byte(content))
}

// RunBufferedRegistration allows to run a registration with a buffered writer
func (e FormattedLogEntry) RunBufferedRegistration(buf BufferedLogEntryWriter, quit chan bool) {
	if e.deploymentID == "" {
		log.Panic("The deploymentID parameter must be filled")
	}

	buf.run(quit, e)
}

func (e FormattedLogEntry) generateKey() string {
	return path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "logs", time.Now().Format(time.RFC3339Nano))
}

func (e FormattedLogEntry) generateValue() []byte {
	// Check content max allowed size
	if len(e.content) > contentMaxAllowedValueSize {
		log.Printf("The max allowed size has been reached: truncation will be done on log content from %d to %d bytes", len(e.content), contentMaxAllowedValueSize)
		e.content = e.content[:contentMaxAllowedValueSize]
	}
	// For presentation purpose, the formatted log entry is cast to flat map
	flatMap := e.toFlatMap()
	b, err := json.Marshal(flatMap)
	if err != nil {
		log.Printf("Failed to marshal entry [%+v]: due to error:%+v", e, err)
	}
	// log the entry in stdout/stderr
	e.log(flatMap)
	return b
}

func (e FormattedLogEntry) toFlatMap() map[string]interface{} {
	flatMap := make(map[string]interface{})

	// NewLogEntry main attributes from FormattedLogEntry
	flatMap["DeploymentID"] = e.deploymentID
	flatMap["Level"] = e.level.String()
	flatMap["Content"] = string(e.content)

	// Add Timestamp
	flatMap["Timestamp"] = getTimestamp()

	// NewLogEntry additional info
	for k, v := range e.additionalInfo {
		flatMap[k.String()] = v
	}
	return flatMap
}

// Log the entry in stdout/stderr with the following format :[Timestamp][Level][DeploymentID][WorkflowID][ExecutionID][NodeID][InstanceID][InterfaceID][OperationID][TypeID][Content]
func (e FormattedLogEntry) log(flat map[string]interface{}) {
	var str string
	sliceOfKeys := []string{"Timestamp", "Level", "DeploymentID", WorkFlowID.String(), ExecutionID.String(), NodeID.String(), InstanceID.String(), InterfaceID.String(), OperationID.String(), TypeID.String(), "Content"}
	for _, k := range sliceOfKeys {
		if val, ok := flat[k].(string); ok {
			str += "[" + val + "]"
		}

	}
	// Log are only displayed in DEBUG mode
	log.Debugln(str)
}

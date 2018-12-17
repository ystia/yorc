// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package events

//go:generate go-enum -f=log_entries.go

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
)

// LogEntry is the log entry representation
type LogEntry struct {
	level          LogLevel
	deploymentID   string
	additionalInfo LogOptionalFields
	content        []byte
	timestamp      time.Time
}

// LogEntryDraft is a partial LogEntry with only optional fields.
// It has to be completed with level and deploymentID
type LogEntryDraft struct {
	additionalInfo LogOptionalFields
}

// LogOptionalFields are log's additional info
type LogOptionalFields map[FieldType]interface{}

// FieldType is allowed/expected additional info types
type FieldType int

const (
	// WorkFlowID is the field type representing the workflow ID in log entry
	WorkFlowID FieldType = iota

	// ExecutionID is the field type representing the execution ID in log entry
	ExecutionID

	// NodeID is the field type representing the node ID in log entry
	NodeID

	// InstanceID is the field type representing the instance ID in log entry
	InstanceID

	// InterfaceName is the field type representing the interface ID in log entry
	InterfaceName

	// OperationName is the field type representing the operation ID in log entry
	OperationName

	// TypeID is the field type representing the type ID in log entry
	TypeID

	// TaskExecutionID is the field type representing the task execution ID in log entry
	TaskExecutionID
)

// String allows to stringify the field type enumeration in JSON standard
func (ft FieldType) String() string {
	switch ft {
	case WorkFlowID:
		return "workflowId"
	case ExecutionID:
		return "executionId"
	case NodeID:
		return "nodeId"
	case InstanceID:
		return "instanceId"
	case InterfaceName:
		return "interfaceName"
	case OperationName:
		return "operationName"
	case TypeID:
		return "type"
	case TaskExecutionID:
		return "alienTaskId"
	}
	return ""
}

// Max allowed storage size in Consul kv for value is Equal to 512 Kb
// We approximate all data except the content value to be equal to 1Kb
const contentMaxAllowedValueSize int = 511 * 1000

// LogLevel x ENUM(
// INFO,
// DEBUG,
// WARN,
// ERROR
// )
type LogLevel int

// SimpleLogEntry allows to return a LogEntry instance with log level and deploymentID
func SimpleLogEntry(level LogLevel, deploymentID string) *LogEntry {
	return &LogEntry{
		level:        level,
		deploymentID: deploymentID,
	}
}

// WithOptionalFields allows to return a LogEntry instance with additional fields
//
// Deprecated: Use events.WithContextOptionalFields() instead
func WithOptionalFields(fields LogOptionalFields) *LogEntryDraft {
	info := make(LogOptionalFields, len(fields))
	fle := &LogEntryDraft{additionalInfo: info}
	for k, v := range fields {
		info[k] = v
	}

	return fle
}

// WithContextOptionalFields allows to return a LogEntry instance with additional fields comming from the context
func WithContextOptionalFields(ctx context.Context) *LogEntryDraft {
	lof, _ := FromContext(ctx)
	return WithOptionalFields(lof)
}

// NewLogEntry allows to build a log entry from a draft
func (e LogEntryDraft) NewLogEntry(level LogLevel, deploymentID string) *LogEntry {
	return &LogEntry{
		level:          level,
		deploymentID:   deploymentID,
		additionalInfo: e.additionalInfo,
	}
}

// Register allows to register a log entry with byte array content
func (e LogEntry) Register(content []byte) {
	if len(content) == 0 {
		log.Panic("The content parameter must be filled")
	}
	if e.deploymentID == "" {
		log.Panic("The deploymentID parameter must be filled")
	}
	e.content = content

	// Get the timestamp
	e.timestamp = time.Now()

	// Handle TaskExecutionID field which needs to be enrich with instanceID
	inst, existInst := e.additionalInfo[InstanceID]
	id, existID := e.additionalInfo[TaskExecutionID]
	if existInst && existID {
		e.additionalInfo[TaskExecutionID] = fmt.Sprintf("%s-%s", id, inst)
	}

	// Get the value to store and the flat log entry representation to log entry
	val, flat := e.generateValue()
	err := consulutil.StoreConsulKey(e.generateKey(), val)
	if err != nil {
		log.Printf("Failed to register log in consul for entry:%+v due to error:%+v", e, err)
	}

	// log the entry in stdout/stderr in DEBUG mode
	// Log are only displayed in DEBUG mode
	log.Debugln(FormatLog(flat))
}

// RegisterAsString allows to register a log entry with string content
func (e LogEntry) RegisterAsString(content string) {
	e.Register([]byte(content))
}

// Registerf allows to register a log entry with formats
// according to a format specifier.
//
// This is basically a convenient function around RegisterAsString(fmt.Sprintf()).
func (e LogEntry) Registerf(format string, a ...interface{}) {
	e.RegisterAsString(fmt.Sprintf(format, a...))
}

// RunBufferedRegistration allows to run a registration with a buffered writer
func (e LogEntry) RunBufferedRegistration(buf BufferedLogEntryWriter, quit chan bool) {
	if e.deploymentID == "" {
		log.Panic("The deploymentID parameter must be filled")
	}

	buf.run(quit, e)
}

func (e LogEntry) generateKey() string {
	// time.RFC3339Nano is needed for ConsulKV key value precision
	return path.Join(consulutil.LogsPrefix, e.deploymentID, e.timestamp.Format(time.RFC3339Nano))
}

func (e LogEntry) generateValue() ([]byte, map[string]interface{}) {
	// Check content max allowed size
	if len(e.content) > contentMaxAllowedValueSize {
		log.Printf("The max allowed size has been reached: truncation will be done on log content from %d to %d bytes", len(e.content), contentMaxAllowedValueSize)
		e.content = e.content[:contentMaxAllowedValueSize]
	}
	// For presentation purpose, the log entry is cast to flat map
	flat := e.toFlatMap()
	b, err := json.Marshal(flat)
	if err != nil {
		log.Printf("Failed to marshal entry [%+v]: due to error:%+v", e, err)
	}
	return b, flat
}

func (e LogEntry) toFlatMap() map[string]interface{} {
	flatMap := make(map[string]interface{})

	// NewLogEntry main attributes from LogEntry
	flatMap["deploymentId"] = e.deploymentID
	flatMap["level"] = e.level.String()
	flatMap["content"] = string(e.content)
	// Keeping a nanosecond precision for timestamps, as it is done for keys
	// generation, so that a client application performing a timestamp sort on
	// a slice of n logs will get exactly the same order as the log insertion
	// order
	flatMap["timestamp"] = e.timestamp.Format(time.RFC3339Nano)

	// NewLogEntry additional info
	for k, v := range e.additionalInfo {
		flatMap[k.String()] = v
	}
	return flatMap
}

// FormatLog allows to format the flat map log representation in the following format :[Timestamp][Level][DeploymentID][WorkflowID][ExecutionID][TaskExecutionID][NodeID][InstanceID][InterfaceName][OperationName][TypeID]Content
func FormatLog(flat map[string]interface{}) string {
	var str string
	sliceOfKeys := []string{"timestamp", "level", "deploymentId", WorkFlowID.String(), ExecutionID.String(), TaskExecutionID.String(), NodeID.String(), InstanceID.String(), InterfaceName.String(), OperationName.String(), TypeID.String(), "content"}
	for _, k := range sliceOfKeys {
		if val, ok := flat[k].(string); ok {
			if k != "content" {
				str += "[" + val + "]"
			} else {
				str += val
			}
		} else {
			str += "[]"
		}

	}
	return str
}

// contextKey is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type contextKey int

// logOptFieldsKey is the key for events.LogOptionalFields values in Contexts. It is
// unexported; clients use events.NewContext and events.FromContext
// instead of using this key directly.
var logOptFieldsKey contextKey

// NewContext returns a new Context that carries value logOptFields.
func NewContext(ctx context.Context, logOptFields LogOptionalFields) context.Context {
	return context.WithValue(ctx, logOptFieldsKey, logOptFields)
}

// FromContext returns a copy of the LogOptionalFields value stored in ctx, if any.
func FromContext(ctx context.Context) (LogOptionalFields, bool) {
	var result LogOptionalFields
	if ctx == nil {
		return result, false
	}
	lof, ok := ctx.Value(logOptFieldsKey).(LogOptionalFields)
	if ok {
		result = make(LogOptionalFields, len(lof))
		for k, v := range lof {
			result[k] = v
		}
	}
	return result, ok
}

// AddLogOptionalFields adds given log optional fields to existing one in the given context if any
//
// Existing fields are overwritten in case of collision
func AddLogOptionalFields(ctx context.Context, logOptFields LogOptionalFields) context.Context {
	existing, ok := FromContext(ctx)
	if !ok {
		existing = make(LogOptionalFields)
	}
	for k, v := range logOptFields {
		existing[k] = v
	}
	return NewContext(ctx, existing)
}

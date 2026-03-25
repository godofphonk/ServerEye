package protocol

// FieldKind enumerates the supported field types inside a command payload schema.
type FieldKind string

const (
	FieldKindString FieldKind = "string"
	FieldKindInt    FieldKind = "int"
	FieldKindFloat  FieldKind = "float"
	FieldKindBool   FieldKind = "bool"
	FieldKindArray  FieldKind = "array"
	FieldKindObject FieldKind = "object"
)

// FieldSchema describes constraints for a single payload field.
type FieldSchema struct {
	Kind          FieldKind
	Required      bool
	MaxLen        int      // max UTF-8 characters for string fields (0 = unlimited)
	MinLen        int      // min UTF-8 characters for string fields
	MaxSize       int      // max number of elements for array fields (0 = unlimited)
	Min           float64  // min value for numeric fields
	Max           float64  // max value for numeric fields
	HasRange      bool     // set to true when Min/Max should be enforced
	AllowedValues []string // if non-empty, the string value must be one of these
}

// CommandSchema describes the expected payload for a specific command type.
type CommandSchema struct {
	// Fields maps field names to their validation rules.
	Fields map[string]FieldSchema
	// AllowExtra permits fields that are not listed in Fields.
	AllowExtra bool
}

// Schemas maps every known MessageType to its CommandSchema.
// Commands that accept no parameters have an empty Fields map.
var Schemas = map[MessageType]CommandSchema{
	// Read-only queries without parameters
	TypeGetCPUTemp:    {Fields: map[string]FieldSchema{}, AllowExtra: false},
	TypeGetSystemInfo: {Fields: map[string]FieldSchema{}, AllowExtra: false},
	TypeGetMemoryInfo: {Fields: map[string]FieldSchema{}, AllowExtra: false},
	TypeGetUptime:     {Fields: map[string]FieldSchema{}, AllowExtra: false},
	TypePing:          {Fields: map[string]FieldSchema{}, AllowExtra: false},

	// Disk info – optional mount path filter
	TypeGetDiskInfo: {
		Fields: map[string]FieldSchema{
			"path": {Kind: FieldKindString, Required: false, MaxLen: 4096},
		},
		AllowExtra: false,
	},

	// Container list – optional filter string
	TypeGetContainers: {
		Fields: map[string]FieldSchema{
			"filter": {Kind: FieldKindString, Required: false, MaxLen: 256},
		},
		AllowExtra: false,
	},

	// Process list – optional limit
	TypeGetProcesses: {
		Fields: map[string]FieldSchema{
			"limit": {Kind: FieldKindInt, Required: false, HasRange: true, Min: 1, Max: 1000},
		},
		AllowExtra: false,
	},

	// Network info – optional interface name filter
	TypeGetNetworkInfo: {
		Fields: map[string]FieldSchema{
			"interface": {Kind: FieldKindString, Required: false, MaxLen: 64},
		},
		AllowExtra: false,
	},

	// Container lifecycle actions
	TypeStartContainer: {
		Fields: map[string]FieldSchema{
			"container_id":   {Kind: FieldKindString, Required: false, MaxLen: 128},
			"container_name": {Kind: FieldKindString, Required: false, MaxLen: 256},
		},
		AllowExtra: false,
	},
	TypeStopContainer: {
		Fields: map[string]FieldSchema{
			"container_id":   {Kind: FieldKindString, Required: false, MaxLen: 128},
			"container_name": {Kind: FieldKindString, Required: false, MaxLen: 256},
		},
		AllowExtra: false,
	},
	TypeRestartContainer: {
		Fields: map[string]FieldSchema{
			"container_id":   {Kind: FieldKindString, Required: false, MaxLen: 128},
			"container_name": {Kind: FieldKindString, Required: false, MaxLen: 256},
		},
		AllowExtra: false,
	},
	TypeRemoveContainer: {
		Fields: map[string]FieldSchema{
			"container_id":   {Kind: FieldKindString, Required: false, MaxLen: 128},
			"container_name": {Kind: FieldKindString, Required: false, MaxLen: 256},
		},
		AllowExtra: false,
	},

	// Create container
	TypeCreateContainer: {
		Fields: map[string]FieldSchema{
			"image":       {Kind: FieldKindString, Required: true, MinLen: 1, MaxLen: 512},
			"name":        {Kind: FieldKindString, Required: false, MaxLen: 256},
			"ports":       {Kind: FieldKindObject, Required: false},
			"environment": {Kind: FieldKindObject, Required: false},
			"volumes":     {Kind: FieldKindObject, Required: false},
		},
		AllowExtra: false,
	},

	// Update agent
	TypeUpdateAgent: {
		Fields: map[string]FieldSchema{
			"version": {Kind: FieldKindString, Required: true, MinLen: 1, MaxLen: 64},
		},
		AllowExtra: false,
	},
}

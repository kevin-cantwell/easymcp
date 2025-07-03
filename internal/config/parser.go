package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	yaml "gopkg.in/yaml.v3"
)

// Config wraps a list of tools
type Config struct {
	Tools []Tool `yaml:"tools"`
}

// Tool defines a namespaced command
type Tool struct {
	Namespace   string  `yaml:"namespace"` // Namespace for the tool, e.g. "demo" or "util"
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Run         Command `yaml:"run"` // Optional command to run
	Input       []Input `yaml:"input"`
	Output      Output  `yaml:"output"`
}

// Input defines a single argument for a tool
type Input struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"` // JSON schema type (One of: "string", "integer", "number", "boolean")
	Description string `yaml:"description"`
	Enum        []any  `yaml:"enum,omitempty"` // Optional enum values
	Required    bool   `yaml:"required"`
}

type Output struct {
	Format string `yaml:"format"` // Output format (One of: "text", "json")
}

type Command struct {
	Cmd  string   `yaml:"cmd"`  // Command to run
	Args []string `yaml:"args"` // Arguments to the command
}

func (t *Tool) InputSchema() (*jsonschema.Schema, error) {
	fields := make([]reflect.StructField, len(t.Input))
	for i, arg := range t.Input {
		name := arg.Name
		if name == "" {
			return nil, errors.New("argument name cannot be empty")
		}
		if len(name) > 64 {
			return nil, errors.New("argument name cannot exceed 64 characters")
		}
		var typ reflect.Type
		switch arg.Type {
		case "string":
			typ = reflect.TypeOf("")
		case "integer":
			typ = reflect.TypeOf(int64(0))
		case "number":
			typ = reflect.TypeOf(float64(0))
		case "boolean":
			typ = reflect.TypeOf(true)
		default:
			return nil, errors.New("invalid argument type: " + arg.Type + ", must be one of: string, integer, number, boolean")
		}
		for j, enum := range arg.Enum {
			switch typ.Kind() {
			case reflect.String:
				arg.Enum[j] = fmt.Sprintf("%v", enum) // Convert to string representation
			case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
				n, err := strconv.ParseInt(fmt.Sprintf("%v", enum), 10, 64) // Ensure integer values are parsed correctly
				if err != nil {
					return nil, fmt.Errorf("enum value '%v' is not a valid integer: %v", enum, err)
				}
				arg.Enum[j] = n
			case reflect.Float64:
				f, err := strconv.ParseFloat(fmt.Sprintf("%v", enum), 64) // Ensure numeric values are parsed correctly
				if err != nil {
					return nil, fmt.Errorf("enum value '%v' is not a valid number: %v", enum, err)
				}
				arg.Enum[j] = f
			case reflect.Bool:
				b, err := strconv.ParseBool(fmt.Sprintf("%v", enum)) // Ensure boolean values are parsed correctly
				if err != nil {
					return nil, fmt.Errorf("enum value '%v' is not a valid boolean: %v", enum, err)
				}
				arg.Enum[j] = b
			default:
				return nil, errors.New("enum value '" + fmt.Sprint(enum) + "' type does not match argument type: " + arg.Type)
			}
		}
		field := reflect.StructField{
			Name: capitalize(name),
			Type: typ,
			Tag:  reflect.StructTag(`json:"` + name + `"`),
		}
		fields[i] = field

	}

	dynamicType := reflect.StructOf(fields)
	schema, err := jsonschema.ForType(dynamicType)
	if err != nil {
		return nil, err
	}

	// Update properties schema based on tool arguments
	for _, arg := range t.Input {
		prop := schema.Properties[arg.Name]
		if prop == nil {
			return nil, fmt.Errorf("missing schema property for argument: %s", arg.Name)
		}
		prop.Description = arg.Description
		if len(arg.Enum) > 0 {
			prop.Enum = arg.Enum
		}
		if !slices.Contains(prop.Required, arg.Name) {
			schema.Required = append(schema.Required, arg.Name)
		}
	}

	return schema, nil
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// Load reads and parses the YAML config file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

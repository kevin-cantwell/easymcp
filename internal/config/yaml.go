package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// MapItem represents a single key-value pair, preserving order.
// Key is the YAML map key, Value is the decoded value of type T.
type MapItem[T any] struct {
	Key   string
	Value T
}

// MapSlice is a slice of MapItem preserving the order of items in a YAML map.
// It supports unmarshaling from both mapping and sequence nodes.
type MapSlice[T any] []MapItem[T]

// UnmarshalYAML implements the yaml.Unmarshaler interface for MapSlice.
// It handles both mapping nodes (preserving the order of Content pairs)
// and sequence nodes of one-item mappings.
func (ms *MapSlice[T]) UnmarshalYAML(node *yaml.Node) error {
	// Temporary slice to collect items
	var items []MapItem[T]

	switch node.Kind {
	case yaml.MappingNode:
		// Content: [keyNode, valNode, keyNode, valNode, ...]
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]

			var v T
			if err := valNode.Decode(&v); err != nil {
				return fmt.Errorf("failed to decode value for key '%s': %w", keyNode.Value, err)
			}

			items = append(items, MapItem[T]{Key: keyNode.Value, Value: v})
		}

	case yaml.SequenceNode:
		// Sequence of single-key mappings: [{key1: val1}, {key2: val2}, ...]
		for _, entry := range node.Content {
			if entry.Kind != yaml.MappingNode || len(entry.Content) != 2 {
				return fmt.Errorf("invalid sequence entry for MapSlice: line %d, column %d", entry.Line, entry.Column)
			}
			keyNode := entry.Content[0]
			valNode := entry.Content[1]

			var v T
			if err := valNode.Decode(&v); err != nil {
				return fmt.Errorf("failed to decode sequence entry value for key '%s': %w", keyNode.Value, err)
			}

			items = append(items, MapItem[T]{Key: keyNode.Value, Value: v})
		}

	default:
		return fmt.Errorf("unexpected YAML node kind %d for MapSlice", node.Kind)
	}

	*ms = items
	return nil
}

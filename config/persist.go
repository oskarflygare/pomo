package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

// SaveDuration sets the duration of a task type in the loaded config
// and writes it to the config file in use, creating the per-user
// config file if none exists. Other keys and comments are preserved.
func SaveDuration(taskType TaskType, duration time.Duration) error {
	path := viper.ConfigFileUsed()
	if path == "" {
		dir, err := getConfigDir()
		if err != nil {
			return err
		}
		path = filepath.Join(dir, ConfigFile)
	}

	key := "work"
	if taskType == BreakTask {
		key = "break"
	}

	if err := writeDuration(path, key, duration); err != nil {
		return err
	}

	taskType.GetTask().Duration = duration
	viper.Set(key+".duration", duration)

	return nil
}

// FormatDuration formats a duration the way it is written in pomo.yaml (e.g. 25m, 1h30m).
func FormatDuration(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = strings.TrimSuffix(s, "0s")
	}
	if strings.HasSuffix(s, "h0m") {
		s = strings.TrimSuffix(s, "0m")
	}
	return s
}

// updates <key>.duration in the YAML file at path
func writeDuration(path, key string, duration time.Duration) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("could not parse %s: %w", path, err)
	}

	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("could not update %s: top level is not a mapping", path)
	}

	formatted := FormatDuration(duration)

	// edit the existing value in place to keep the file's formatting intact
	if task := findValue(root, key); task != nil && task.Kind == yaml.MappingNode {
		if value := findValue(task, "duration"); value != nil && value.Kind == yaml.ScalarNode {
			if edited, ok := replaceScalar(data, value, formatted); ok {
				return os.WriteFile(path, edited, 0o644)
			}
		}
	}

	// otherwise add the missing keys and re-encode the document
	task := mappingValue(root, key, yaml.MappingNode)
	value := mappingValue(task, "duration", yaml.ScalarNode)
	value.Tag = "!!str"
	value.Value = formatted

	var out strings.Builder
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(out.String()), 0o644)
}

// replaces the source text of a single-line scalar node with value
func replaceScalar(data []byte, node *yaml.Node, value string) ([]byte, bool) {
	tokenLen := len(node.Value)
	switch node.Style {
	case 0:
	case yaml.DoubleQuotedStyle, yaml.SingleQuotedStyle:
		tokenLen += 2
	default:
		return nil, false
	}

	lines := strings.SplitAfter(string(data), "\n")
	if node.Line < 1 || node.Line > len(lines) {
		return nil, false
	}

	line := lines[node.Line-1]
	start := node.Column - 1
	if start < 0 || start+tokenLen > len(line) {
		return nil, false
	}

	lines[node.Line-1] = line[:start] + value + line[start+tokenLen:]
	return []byte(strings.Join(lines, "")), true
}

// returns the value node for key in mapping, or nil if missing
func findValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

// returns the value node for key in mapping, appending one of the given kind if missing
func mappingValue(mapping *yaml.Node, key string, kind yaml.Kind) *yaml.Node {
	if value := findValue(mapping, key); value != nil {
		return value
	}

	value := &yaml.Node{Kind: kind}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		value,
	)

	return value
}

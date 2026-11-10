package world

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

func validateWorldJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	if root == nil {
		return fmt.Errorf("world definition must be a JSON object")
	}

	rooms, exists := root["rooms"]
	if !exists {
		return fmt.Errorf("required field world.rooms is missing")
	}

	sections := []struct {
		name     string
		required []string
	}{
		{name: "rooms", required: []string{"id", "name"}},
		{name: "items", required: []string{"id", "name", "type"}},
		{name: "npcs", required: []string{"id", "name", "combat_stats"}},
		{name: "quests", required: []string{"id", "name", "giver_npc_id"}},
	}

	for _, section := range sections {
		rawSection, sectionExists := root[section.name]
		if !sectionExists {
			continue
		}
		if section.name == "rooms" {
			rawSection = rooms
		}

		entities, err := jsonObjectFields(
			rawSection,
			"world."+section.name,
		)
		if err != nil {
			return err
		}

		entityIDs := make([]string, 0, len(entities))
		for entityID := range entities {
			entityIDs = append(entityIDs, entityID)
		}
		sort.Strings(entityIDs)

		for _, entityID := range entityIDs {
			path := fmt.Sprintf("world.%s[%q]", section.name, entityID)
			entity, err := jsonObjectFields(entities[entityID], path)
			if err != nil {
				return err
			}
			if err := requireJSONFields(entity, path, section.required...); err != nil {
				return err
			}

			if section.name == "npcs" {
				stats, err := jsonObjectFields(entity["combat_stats"], path+".combat_stats")
				if err != nil {
					return err
				}
				if err := requireJSONFields(
					stats,
					path+".combat_stats",
					"hp",
					"max_hp",
					"attack",
					"defense",
					"speed",
				); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}

	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := token.(string)
			if !ok {
				return fmt.Errorf("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}

			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err

	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err

	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func jsonObjectFields(raw json.RawMessage, path string) (map[string]json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return nil, fmt.Errorf("%s must be a JSON object", path)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return fields, nil
}

func requireJSONFields(fields map[string]json.RawMessage, path string, names ...string) error {
	for _, name := range names {
		value, exists := fields[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("required field %s.%s is missing or null", path, name)
		}
	}
	return nil
}

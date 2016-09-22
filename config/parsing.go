package config

import (
	"encoding/json"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func getFormat(fn string) (amboy.Format, error) {
	if strings.HasSuffix(fn, ".yaml") || strings.HasSuffix(fn, ".yml") {
		return amboy.YAML, nil
	} else if strings.HasSuffix(fn, ".json") {
		return amboy.JSON, nil
	}

	return nil, errors.Errorf("greenbay does not support configuration format for file %s", fn)
}

func getJSONFormatedConfig(format amboy.Format, data []byte) ([]byte, error) {
	if format == amboy.JSON {
		return data, nil
	} else if format == amboy.YAML {
		// the yaml package does not include a way to do the kind of
		// delayed parsing that encoding/json permits, so we cycle
		// into a map and then through the JSON parser itself.
		intermediateOut := make(map[string]interface{})
		err = yaml.Unmarshal(data, intermediateOut)
		if err != nil {
			return nil, errors.Wrap(err, "problem parsing yaml config")
		}

		data, err = json.Marshal(intermediateOut)
		if err != nil {
			return nil, errors.Wrap(err, "problem converting yaml to intermediate json")
		}

		return data, nil
	}

	return nil, errors.Errorf("format %s is not supported", format)
}

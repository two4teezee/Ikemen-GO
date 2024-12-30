package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

// -------------------------------------------------------------------
// Helper Functions
// -------------------------------------------------------------------

// queryPart represents a single part of a query path
type queryPart struct {
	name  string
	index *string
}

// parseQueryPath parses a query string into a slice of queryPart
func parseQueryPath(query string) []queryPart {
	var parts []queryPart
	for _, part := range strings.Split(query, ".") {
		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			name := part[:strings.Index(part, "[")]
			index := part[strings.Index(part, "[")+1 : len(part)-1]
			parts = append(parts, queryPart{name: name, index: &index})
		} else {
			parts = append(parts, queryPart{name: part, index: nil})
		}
	}
	return parts
}

// getValueFromPatternMap checks if the struct has a map field with ini:"map:REGEX"
func getValueFromPatternMap(v reflect.Value, partName string) (bool, reflect.Value) {
	if v.Kind() != reflect.Struct {
		return false, reflect.Value{}
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		iniTag := field.Tag.Get("ini")
		if iniTag == "" || field.PkgPath != "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(iniTag), "map:") {
			pattern := iniTag[4:]
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}
			// Check if partName matches the map regex
			if re.MatchString(partName) {
				fieldVal := v.Field(i)
				if fieldVal.Kind() == reflect.Map && fieldVal.Type().Key().Kind() == reflect.String {
					if fieldVal.IsNil() {
						fieldVal.Set(reflect.MakeMap(fieldVal.Type()))
					}
					// Convert to lower or keep original
					mapKey := reflect.ValueOf(strings.ToLower(partName))
					mapItem := fieldVal.MapIndex(mapKey)
					if !mapItem.IsValid() {
						// Create a new entry
						elemType := fieldVal.Type().Elem()
						var newVal reflect.Value
						if elemType.Kind() == reflect.Ptr {
							newVal = reflect.New(elemType.Elem())
						} else {
							newVal = reflect.New(elemType).Elem()
						}
						applyDefaultsToValue(newVal)
						fieldVal.SetMapIndex(mapKey, newVal)
						mapItem = fieldVal.MapIndex(mapKey)
					}
					return true, mapItem
				}
			}
		}
	}
	return false, reflect.Value{}
}

// -------------------------------------------------------------------
// Field Retrieval Functions
// -------------------------------------------------------------------

// findFieldByINITag recursively searches for a struct field with a matching INI tag
func findFieldByINITag(v reflect.Value, tag string) (reflect.Value, bool) {
	return findFieldByINITagRecursive(v, v.Type(), tag)
}

// findFieldByINITagRecursive is a helper function for findFieldByINITag
func findFieldByINITagRecursive(val reflect.Value, typ reflect.Type, tag string) (reflect.Value, bool) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fVal := val.Field(i)
		iniTag := field.Tag.Get("ini")
		if iniTag != "" && strings.EqualFold(iniTag, tag) && field.PkgPath == "" {
			return fVal, true
		}
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if rv, found := findFieldByINITagRecursive(fVal, field.Type, tag); found {
				return rv, true
			}
		}
	}
	return reflect.Value{}, false
}

// findMapFieldWithTag searches for a map field in a struct with a specific INI tag
func findMapFieldWithTag(v reflect.Value, tag string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if strings.EqualFold(field.Tag.Get("ini"), tag) && field.Type.Kind() == reflect.Map {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// findDefaultField returns the first struct field without an INI tag
func findDefaultField(v reflect.Value) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("ini") == "" {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// -------------------------------------------------------------------
// Default Tag Applying Functions
// -------------------------------------------------------------------

// applyDefaultsToValue applies default-tagged values to any zero fields.
// This works recursively on structs, pointers, slices, and arrays.
func applyDefaultsToValue(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return
		}
		applyDefaultsToValue(v.Elem())

	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldType := t.Field(i)

			if fieldType.PkgPath != "" {
				continue
			}

			applyDefaultsToValue(fieldVal)

			defTag := fieldType.Tag.Get("default")
			if defTag != "" {
				if isZeroValue(fieldVal) {
					setDefaultFieldValue(fieldVal, fieldType, defTag)
				}
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			applyDefaultsToValue(v.Index(i))
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			applyDefaultsToValue(val)
		}
	default:
		// Base kind: string, int, float, bool, etc. -> no recursion
	}
}

// setDefaultFieldValue tries to parse `defTag` and set it into fieldVal
func setDefaultFieldValue(fieldVal reflect.Value, fieldType reflect.StructField, defTag string) {
	if !fieldVal.CanSet() {
		return
	}
	kind := fieldVal.Kind()

	switch kind {
	case reflect.String:
		fieldVal.SetString(defTag)
	case reflect.Bool:
		if parsed, err := strconv.ParseBool(defTag); err == nil {
			fieldVal.SetBool(parsed)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if parsed, err := strconv.ParseInt(defTag, 10, 64); err == nil {
			fieldVal.SetInt(parsed)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if parsed, err := strconv.ParseUint(defTag, 10, 64); err == nil {
			fieldVal.SetUint(parsed)
		}
	case reflect.Float32, reflect.Float64:
		if parsed, err := strconv.ParseFloat(defTag, 64); err == nil {
			fieldVal.SetFloat(parsed)
		}
	case reflect.Array:
		parts := strings.Split(defTag, ",")
		for i := 0; i < fieldVal.Len() && i < len(parts); i++ {
			elem := fieldVal.Index(i)
			setDefaultFieldValue(elem, fieldType, strings.TrimSpace(parts[i]))
		}
	case reflect.Slice:
		parts := strings.Split(defTag, ",")
		newSlice := reflect.MakeSlice(fieldVal.Type(), 0, len(parts))
		for _, part := range parts {
			elem := reflect.New(fieldVal.Type().Elem()).Elem()
			setDefaultFieldValue(elem, fieldType, strings.TrimSpace(part))
			newSlice = reflect.Append(newSlice, elem)
		}
		fieldVal.Set(newSlice)
	case reflect.Struct:
		parts := strings.Split(defTag, ",")
		for i := 0; i < fieldVal.NumField() && i < len(parts); i++ {
			fv := fieldVal.Field(i)
			setDefaultFieldValue(fv, fieldType, strings.TrimSpace(parts[i]))
		}
	}
}

// isZeroValue returns true if the reflect.Value is a zero value
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZeroValue(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		z := reflect.Zero(v.Type()).Interface()
		return reflect.DeepEqual(v.Interface(), z)
	}
}

// -------------------------------------------------------------------
// Value Assignment Functions
// -------------------------------------------------------------------

// setFieldValue assigns a value to a field based on its type
func setFieldValue(fieldVal reflect.Value, value interface{}, keyPath string) error {
	if value == nil {
		switch fieldVal.Kind() {
		case reflect.Map:
			return nil
		case reflect.Slice, reflect.Array:
			fieldVal.Set(reflect.MakeSlice(fieldVal.Type(), 0, 0))
			return nil
		case reflect.String:
			fieldVal.SetString("")
			return nil
		case reflect.Bool:
			fieldVal.SetBool(false)
			return nil
		case reflect.Int, reflect.Int32, reflect.Int64:
			fieldVal.SetInt(0)
			return nil
		case reflect.Float32, reflect.Float64:
			fieldVal.SetFloat(0.0)
			return nil
		default:
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
	}

	if arr, ok := value.([]string); ok {
		switch fieldVal.Kind() {
		case reflect.Slice:
			newSlice := reflect.MakeSlice(fieldVal.Type(), 0, len(arr))
			for _, v := range arr {
				elem := reflect.New(fieldVal.Type().Elem()).Elem()
				if err := setFieldValue(elem, v, keyPath); err != nil {
					return err
				}
				newSlice = reflect.Append(newSlice, elem)
			}
			fieldVal.Set(newSlice)
			return nil
		case reflect.Array:
			for i := 0; i < fieldVal.Len() && i < len(arr); i++ {
				elem := fieldVal.Index(i)
				if err := setFieldValue(elem, arr[i], keyPath); err != nil {
					return err
				}
			}
			return nil
		default:
			if len(arr) > 0 {
				return setFieldValue(fieldVal, arr[0], keyPath)
			}
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
	}

	strVal, ok := value.(string)
	if !ok {
		strVal = fmt.Sprintf("%v", value)
	}

	trimmedValue := strings.TrimSpace(strVal)

	fieldKind := fieldVal.Kind()
	if fieldKind == reflect.Ptr {
		if fieldVal.IsNil() {
			fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
			applyDefaultsToValue(fieldVal)
		}
		fieldVal = fieldVal.Elem()
		fieldKind = fieldVal.Kind()
	}

	if trimmedValue == "" {
		switch fieldKind {
		case reflect.String:
			fieldVal.SetString("")
			return nil
		case reflect.Bool:
			fieldVal.SetBool(false)
			return nil
		case reflect.Int, reflect.Int32, reflect.Int64:
			fieldVal.SetInt(0)
			return nil
		case reflect.Float32, reflect.Float64:
			fieldVal.SetFloat(0.0)
			return nil
		case reflect.Slice:
			fieldVal.Set(reflect.MakeSlice(fieldVal.Type(), 0, 0))
			return nil
		default:
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
	}

	switch fieldKind {
	case reflect.Bool:
		parsed, err := strconv.ParseBool(trimmedValue)
		if err != nil {
			return fmt.Errorf("failed to parse bool for '%s': %v", keyPath, err)
		}
		fieldVal.SetBool(parsed)
	case reflect.Slice:
		if strings.Contains(trimmedValue, ",") {
			parts := strings.Split(trimmedValue, ",")
			newSlice := reflect.MakeSlice(fieldVal.Type(), 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				elem := reflect.New(fieldVal.Type().Elem()).Elem()
				if err := setFieldValue(elem, p, keyPath); err != nil {
					return err
				}
				newSlice = reflect.Append(newSlice, elem)
			}
			fieldVal.Set(newSlice)
			return nil
		} else {
			newElem := reflect.New(fieldVal.Type().Elem()).Elem()
			if err := setFieldValue(newElem, trimmedValue, keyPath); err != nil {
				return err
			}
			fieldVal.Set(reflect.Append(fieldVal, newElem))
			return nil
		}
	case reflect.Int32, reflect.Int, reflect.Int64:
		parsed, err := strconv.ParseInt(trimmedValue, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse int: %v", err)
		}
		fieldVal.SetInt(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(trimmedValue, 64)
		if err != nil {
			return fmt.Errorf("failed to parse float: %v", err)
		}
		fieldVal.SetFloat(parsed)
	case reflect.String:
		parsed := strings.Trim(trimmedValue, "\"")
		fieldVal.SetString(parsed)
	case reflect.Array:
		var separator string
		if strings.Contains(trimmedValue, "&") {
			separator = "&"
		} else {
			separator = ","
		}
		parts := strings.Split(trimmedValue, separator)
		for i := 0; i < fieldVal.Len() && i < len(parts); i++ {
			elem := fieldVal.Index(i)
			valStr := strings.TrimSpace(parts[i])
			if err := setFieldValue(elem, valStr, keyPath); err != nil {
				continue
			}
		}
		return nil
	case reflect.Struct:
		defaultField, found := findDefaultField(fieldVal)
		if !found {
			applyDefaultsToValue(fieldVal)
			return nil
		}
		err := setFieldValue(defaultField, trimmedValue, keyPath)
		applyDefaultsToValue(fieldVal)
		return err
	default:
		return fmt.Errorf("unsupported field kind: %s", fieldKind)
	}
	return nil
}

// assignField assigns a value to a struct field based on query parts
func assignField(structPtr interface{}, parts []queryPart, value interface{}) error {
	v := reflect.ValueOf(structPtr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		fmt.Println("Error: structPtr must be a non-nil pointer")
		return fmt.Errorf("structPtr must be a non-nil pointer")
	}

	extractNames := func(parts []queryPart) []string {
		names := make([]string, len(parts))
		for i, part := range parts {
			names[i] = part.name
			if part.index != nil {
				names[i] += "[" + *part.index + "]"
			}
		}
		return names
	}

	assignDirect := func(v reflect.Value, part queryPart, value interface{}) (bool, error) {
		if v.Kind() == reflect.Struct {
			fieldVal, found := findFieldByINITag(v, part.name)
			if found {
				err := setFieldValue(fieldVal, value, strings.Join(extractNames(parts), "."))
				return true, err
			}
		}
		return false, nil
	}

	assignToPatternMap := func(v reflect.Value, lastPartName string, value interface{}, final bool) (bool, reflect.Value, error) {
		if v.Kind() != reflect.Struct {
			return false, v, nil
		}

		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			iniTag := field.Tag.Get("ini")
			if iniTag == "" || field.PkgPath != "" {
				continue
			}

			if strings.HasPrefix(strings.ToLower(iniTag), "map:") {
				pattern := iniTag[4:]
				if pattern == "" {
					continue
				}
				re, err := regexp.Compile(pattern)
				if err != nil {
					continue
				}
				if re.MatchString(lastPartName) {
					fieldVal := v.Field(i)
					if fieldVal.Kind() == reflect.Map && fieldVal.Type().Key().Kind() == reflect.String {
						if fieldVal.IsNil() {
							fieldVal.Set(reflect.MakeMap(fieldVal.Type()))
						}
						mapKey := reflect.ValueOf(strings.ToLower(lastPartName))
						elemType := fieldVal.Type().Elem()

						var newVal reflect.Value
						if elemType.Kind() == reflect.Ptr {
							newVal = reflect.New(elemType.Elem())
						} else {
							newVal = reflect.New(elemType).Elem()
						}
						applyDefaultsToValue(newVal)

						if final {
							if err := setFieldValue(newVal, value, lastPartName); err != nil {
								return false, v, err
							}
						}
						mapElem := fieldVal.MapIndex(mapKey)
						if !mapElem.IsValid() {
							fieldVal.SetMapIndex(mapKey, newVal)
							mapElem = fieldVal.MapIndex(mapKey)
						} else {
							applyDefaultsToValue(mapElem)
						}
						return true, mapElem, nil
					}
				}
			}
		}
		return false, v, nil
	}

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
				applyDefaultsToValue(v)
			}
			v = v.Elem()
		}

		if i == len(parts)-1 {
			if v.Kind() == reflect.Struct {
				assigned, err := assignDirect(v, part, value)
				if err != nil {
					return err
				}
				if assigned {
					return nil
				}

				assigned, _, err = assignToPatternMap(v, part.name, value, true)
				if err != nil {
					return err
				}
				if assigned {
					return nil
				}

				return fmt.Errorf("field '%s' not found in struct or pattern map field", part.name)
			} else if v.Kind() == reflect.Map {
				if value == nil {
					mapKey := reflect.ValueOf(part.name)
					v.SetMapIndex(mapKey, reflect.Value{})
					return nil
				}
				mapKey := reflect.ValueOf(part.name)
				elemType := v.Type().Elem()

				var newVal reflect.Value
				if elemType.Kind() == reflect.Ptr {
					newVal = reflect.New(elemType.Elem())
				} else {
					newVal = reflect.New(elemType).Elem()
				}
				applyDefaultsToValue(newVal)

				if err := setFieldValue(newVal, value, strings.Join(extractNames(parts), ".")); err != nil {
					return err
				}
				v.SetMapIndex(mapKey, newVal)
				return nil
			} else {
				return fmt.Errorf("cannot set value on non-struct and non-map kind '%s'", v.Kind())
			}
		}

		if v.Kind() == reflect.Struct {
			fieldVal, found := findFieldByINITag(v, part.name)
			if found {
				v = fieldVal
				continue
			}

			mapFieldVal, found := findMapFieldWithTag(v, part.name)
			if found && mapFieldVal.Kind() == reflect.Map {
				if i+1 >= len(parts) {
					return fmt.Errorf("expected map key after '%s'", part.name)
				}
				mapKey := parts[i+1].name
				mapKeyVal := reflect.ValueOf(mapKey)
				if mapFieldVal.IsNil() {
					mapFieldVal.Set(reflect.MakeMap(mapFieldVal.Type()))
				}
				elemType := mapFieldVal.Type().Elem()
				mapElem := mapFieldVal.MapIndex(mapKeyVal)
				var elem reflect.Value
				if mapElem.IsValid() {
					elem = mapElem
				} else {
					if elemType.Kind() == reflect.Ptr {
						elem = reflect.New(elemType.Elem())
					} else {
						elem = reflect.New(elemType).Elem()
					}
					applyDefaultsToValue(elem)
					mapFieldVal.SetMapIndex(mapKeyVal, elem)
				}
				v = elem
				i++
				continue
			}

			assigned, newV, err := assignToPatternMap(v, part.name, value, false)
			if err != nil {
				return err
			}
			if assigned {
				v = newV
				continue
			}

			return fmt.Errorf("field '%s' not found as struct or map field", part.name)

		} else if v.Kind() == reflect.Map {
			if i+1 >= len(parts) {
				return fmt.Errorf("expected map key after '%s'", part.name)
			}
			mapKey := part.name
			mapKeyVal := reflect.ValueOf(mapKey)
			if v.IsNil() {
				v.Set(reflect.MakeMap(v.Type()))
			}
			elemType := v.Type().Elem()
			mapElem := v.MapIndex(mapKeyVal)
			var elem reflect.Value
			if mapElem.IsValid() {
				elem = mapElem
			} else {
				if elemType.Kind() == reflect.Ptr {
					elem = reflect.New(elemType.Elem())
				} else {
					elem = reflect.New(elemType).Elem()
				}
				applyDefaultsToValue(elem)
				v.SetMapIndex(mapKeyVal, elem)
			}
			v = elem
			continue
		} else {
			return fmt.Errorf("cannot traverse into non-struct and non-map kind '%s'", v.Kind())
		}
	}

	return nil
}

// updateINIFile updates the INI file based on the struct, query, and value
func updateINIFile(obj interface{}, iniFile *ini.File, query string, value string) error {
	parts := parseQueryPath(query)
	if len(parts) == 0 {
		return fmt.Errorf("invalid query: '%s'", query)
	}

	getSectionAndKeyForPatternMap := func(v reflect.Value, partName string) bool {
		if v.Kind() != reflect.Struct {
			return false
		}
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			iniTag := field.Tag.Get("ini")
			if iniTag == "" || field.PkgPath != "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(iniTag), "map:") {
				pattern := iniTag[4:]
				re, err := regexp.Compile(pattern)
				if err != nil {
					continue
				}
				if re.MatchString(partName) {
					return true
				}
			}
		}
		return false
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	currentValue := v
	currentType := v.Type()

	var sectionName string    // which [section] in the .ini
	var keyNameParts []string // the final "Key.SubKey..." part
	sectionSet := false
	i := 0

	for i < len(parts) {
		part := parts[i]

		// unwrap pointers
		for currentValue.Kind() == reflect.Ptr {
			if currentValue.IsNil() {
				currentValue.Set(reflect.New(currentValue.Type().Elem()))
				applyDefaultsToValue(currentValue)
			}
			currentValue = currentValue.Elem()
			currentType = currentValue.Type()
		}

		switch currentValue.Kind() {
		case reflect.Struct:
			// First, see if we can find a field with ini:"part.name"
			fieldVal, found := findFieldByINITag(currentValue, part.name)
			if found {
				// We found a direct field. We'll record its iniTag as the "section" or "subKey".
				var structField reflect.StructField
				fieldFound := false
				for idx := 0; idx < currentType.NumField(); idx++ {
					f := currentType.Field(idx)
					iniTag := f.Tag.Get("ini")
					if strings.EqualFold(iniTag, part.name) {
						structField = f
						fieldFound = true
						break
					}
					// fallback: check if the field name itself matches
					if !fieldFound && strings.EqualFold(f.Name, part.name) {
						structField = f
						fieldFound = true
					}
				}

				if !fieldFound {
					return fmt.Errorf("field '%s' not found in struct for query '%s'", part.name, query)
				}

				tag := structField.Tag.Get("ini")
				if tag == "" {
					tag = structField.Name
				}

				if !sectionSet {
					sectionName = tag
					sectionSet = true
				} else {
					keyNameParts = append(keyNameParts, tag)
				}

				currentValue = fieldVal
				currentType = currentValue.Type()

				if part.index != nil {
					if currentValue.Kind() == reflect.Array || currentValue.Kind() == reflect.Slice {
						idx, err := strconv.Atoi(*part.index)
						if err != nil || idx < 0 || idx >= currentValue.Len() {
							return fmt.Errorf("invalid index '%s' for field '%s' in query '%s'", *part.index, part.name, query)
						}
						currentValue = currentValue.Index(idx)
						currentType = currentValue.Type()
					} else {
						return fmt.Errorf("field '%s' is not an array or slice in query '%s'", part.name, query)
					}
				}
				i++

			} else {
				// Not a direct field -> check if it's a direct map field with the same tag
				mapFieldVal, mapFound := findMapFieldWithTag(currentValue, part.name)
				if mapFound && mapFieldVal.Kind() == reflect.Map {
					if i+1 >= len(parts) {
						return fmt.Errorf("expected map key after '%s'", part.name)
					}
					if !sectionSet {
						sectionName = part.name
						sectionSet = true
					} else {
						keyNameParts = append(keyNameParts, part.name)
					}
					i++
					mapKey := parts[i].name
					keyNameParts = append(keyNameParts, mapKey)
					i++
					goto finalize

				} else {
					// Now do the pattern-based approach
					foundPattern := getSectionAndKeyForPatternMap(currentValue, part.name)
					if !foundPattern {
						return fmt.Errorf("field '%s' not found as struct or map field", part.name)
					}
					if !sectionSet {
						sectionName = part.name
						sectionSet = true
					} else {
						keyNameParts = append(keyNameParts, part.name)
					}

					i++
					// If there's another part after e.g. Up, that is the actual final key
					if i < len(parts) {
						// e.g. next part is "Up"
						keyNameParts = append(keyNameParts, parts[i].name)
						i++
					}
					goto finalize
				}
			}

		case reflect.Map:
			// If we've stepped into a map, the next part is the map key
			if part.name == "" {
				return fmt.Errorf("expected key for map in query '%s'", query)
			}
			if !sectionSet {
				sectionName = part.name
				sectionSet = true
			} else {
				keyNameParts = append(keyNameParts, part.name)
			}
			i++

		default:
			return fmt.Errorf("unexpected kind '%s' at part '%s'", currentValue.Kind(), part.name)
		}
	}

finalize:
	if len(keyNameParts) == 0 {
		return fmt.Errorf("unable to determine key name from query '%s'", query)
	}

	keyName := strings.Join(keyNameParts, ".")

	section, err := iniFile.GetSection(sectionName)
	if err != nil {
		section, err = iniFile.NewSection(sectionName)
		if err != nil {
			return fmt.Errorf("failed to create section '%s': %v", sectionName, err)
		}
	}

	// If the value is set to nil, remove the key
	if value == "nil" {
		section.DeleteKey(keyName)
		return nil
	}

	key, err := section.GetKey(keyName)
	if err != nil {
		key, err = section.NewKey(keyName, value)
		if err != nil {
			return fmt.Errorf("failed to create key '%s' in section '%s': %v", keyName, sectionName, err)
		}
	} else {
		key.SetValue(value)
	}

	return nil
}

// -------------------------------------------------------------------
// Core Operations
// -------------------------------------------------------------------

// GetValue retrieves a value from the struct based on the query and returns it as an interface{}
func GetValue(structPtr interface{}, query string) (interface{}, error) {
	parts := parseQueryPath(query)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid query: '%s'", query)
	}

	var current reflect.Value
	current = reflect.ValueOf(structPtr)

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return nil, fmt.Errorf("value not set for query: '%s'", query)
			}
			current = current.Elem()
		}

		switch current.Kind() {
		case reflect.Struct:
			// 1) Try normal ini tag
			fieldVal, found := findFieldByINITag(current, part.name)
			if found {
				current = fieldVal
			} else {
				// 2) Try pattern-based map
				matched, mapVal := getValueFromPatternMap(current, part.name)
				if matched {
					current = mapVal
				} else {
					return nil, fmt.Errorf("field '%s' not found in struct or pattern map for query '%s'", part.name, query)
				}
			}

			// if we have an index, check if it's slice/array
			if part.index != nil {
				if current.Kind() == reflect.Array || current.Kind() == reflect.Slice {
					idx, err := strconv.Atoi(*part.index)
					if err != nil || idx < 0 || idx >= current.Len() {
						return nil, fmt.Errorf("invalid index '%s' for field '%s' in query '%s'", *part.index, part.name, query)
					}
					current = current.Index(idx)
				} else {
					return nil, fmt.Errorf("field '%s' is not an array or slice in query '%s'", part.name, query)
				}
			}

		case reflect.Map:
			if part.name == "" {
				return nil, fmt.Errorf("expected key for map in query '%s'", query)
			}
			mapKey := reflect.ValueOf(part.name)
			mapVal := current.MapIndex(mapKey)
			if !mapVal.IsValid() {
				return nil, fmt.Errorf("key '%s' not found in map for query '%s'", part.name, query)
			}
			current = mapVal

			if part.index != nil {
				if current.Kind() == reflect.Array || current.Kind() == reflect.Slice {
					idx, err := strconv.Atoi(*part.index)
					if err != nil || idx < 0 || idx >= current.Len() {
						return nil, fmt.Errorf("invalid index '%s' for key '%s' in query '%s'", *part.index, part.name, query)
					}
					current = current.Index(idx)
				} else {
					return nil, fmt.Errorf("value for key '%s' is not an array or slice in query '%s'", part.name, query)
				}
			}

		default:
			return nil, fmt.Errorf("unsupported kind '%s' in query '%s'", current.Kind(), query)
		}
	}

	switch current.Kind() {
	case reflect.Int32, reflect.Int, reflect.Int64:
		return current.Int(), nil
	case reflect.Float32, reflect.Float64:
		return RoundFloat(current.Float(), 6), nil
	case reflect.String:
		return current.String(), nil
	case reflect.Bool:
		return current.Bool(), nil
	default:
		return current.Interface(), nil
	}
}

// SetValue assigns a value to a struct field based on the query
func SetValue(structPtr interface{}, query string, val interface{}) error {
	parts := parseQueryPath(query)
	if len(parts) == 0 {
		return fmt.Errorf("invalid query: '%s'", query)
	}
	return assignField(structPtr, parts, val)
}

// SetValueUpdate sets a value and updates the INI file accordingly
func SetValueUpdate(obj interface{}, iniFile *ini.File, query string, value interface{}) error {
	err := SetValue(obj, query, value)
	if err != nil {
		return err
	}

	// Convert slices to comma-separated strings
	var valStr string
	switch v := value.(type) {
	case nil:
		valStr = "nil"
	case []string:
		valStr = strings.Join(v, ", ")
	default:
		valStr = fmt.Sprintf("%v", v)
	}

	err = updateINIFile(obj, iniFile, query, valStr)
	if err != nil {
		return err
	}

	return nil
}

// -------------------------------------------------------------------
// Printing Functions
// -------------------------------------------------------------------

// PrintStruct displays the contents of a struct with indentation
func PrintStruct(v interface{}, indent string) {
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			fmt.Printf("%s<nil>\n", indent)
			return
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		fmt.Printf("%s%v\n", indent, val.Interface())
		return
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		fieldName := fieldType.Name

		if tag, ok := fieldType.Tag.Lookup("ini"); ok && tag != "" {
			fieldName = tag
		}

		switch field.Kind() {
		case reflect.Struct:
			fmt.Printf("%s%s:\n", indent, fieldName)
			PrintStruct(field.Interface(), indent+"  ")
		case reflect.Map:
			if field.IsNil() {
				fmt.Printf("%s%s: {}\n", indent, fieldName)
				continue
			}
			fmt.Printf("%s%s:\n", indent, fieldName)
			for _, key := range field.MapKeys() {
				mapValue := field.MapIndex(key)
				if mapValue.Kind() == reflect.String {
					fmt.Printf("%s  %v: \"%v\"\n", indent, key, mapValue.Interface())
				} else {
					fmt.Printf("%s  %v:\n", indent, key)
					PrintStruct(mapValue.Interface(), indent+"    ")
				}
			}
		case reflect.Slice, reflect.Array:
			fmt.Printf("%s%s: [\n", indent, fieldName)
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				PrintStruct(elem.Interface(), indent+"    ")
			}
			fmt.Printf("%s]\n", indent)
		case reflect.String:
			fmt.Printf("%s%s: \"%v\"\n", indent, fieldName, field.Interface())
		default:
			fmt.Printf("%s%s: %v\n", indent, fieldName, field.Interface())
		}
	}
}

// PrintValue displays the value from the struct based on the query
func PrintValue(structPtr interface{}, query string) {
	parts := parseQueryPath(query)
	if len(parts) == 0 {
		fmt.Printf("Invalid query: '%s'\n", query)
		return
	}

	var current reflect.Value
	current = reflect.ValueOf(structPtr)

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				fmt.Printf("Value not set for query: '%s'\n", query)
				return
			}
			current = current.Elem()
		}

		switch current.Kind() {
		case reflect.Struct:
			// 1) Try normal ini tag
			fieldVal, found := findFieldByINITag(current, part.name)
			if found {
				current = fieldVal
			} else {
				// 2) Try pattern-based map
				matched, mapVal := getValueFromPatternMap(current, part.name)
				if matched {
					current = mapVal
				} else {
					fmt.Printf("Field '%s' not found in struct or pattern map for query '%s'\n", part.name, query)
					return
				}
			}

			if part.index != nil {
				if current.Kind() == reflect.Array || current.Kind() == reflect.Slice {
					idx, err := strconv.Atoi(*part.index)
					if err != nil || idx < 0 || idx >= current.Len() {
						fmt.Printf("Invalid index '%s' for field '%s' in query '%s'\n", *part.index, part.name, query)
						return
					}
					current = current.Index(idx)
				} else {
					fmt.Printf("Field '%s' is not an array or slice in query '%s'\n", part.name, query)
					return
				}
			}

		case reflect.Map:
			if part.name == "" {
				fmt.Printf("Expected key for map in query '%s'\n", query)
				return
			}
			mapKey := reflect.ValueOf(part.name)
			mapVal := current.MapIndex(mapKey)
			if !mapVal.IsValid() {
				fmt.Printf("Key '%s' not found in map for query '%s'\n", part.name, query)
				return
			}
			current = mapVal

			if part.index != nil {
				if current.Kind() == reflect.Array || current.Kind() == reflect.Slice {
					idx, err := strconv.Atoi(*part.index)
					if err != nil || idx < 0 || idx >= current.Len() {
						fmt.Printf("Invalid index '%s' for key '%s' in query '%s'\n", *part.index, part.name, query)
						return
					}
					current = current.Index(idx)
				} else {
					fmt.Printf("Value for key '%s' is not an array or slice in query '%s'\n", part.name, query)
					return
				}
			}

		default:
			fmt.Printf("Unsupported kind '%s' in query '%s'\n", current.Kind(), query)
			return
		}
	}

	switch current.Kind() {
	case reflect.Int32, reflect.Int, reflect.Int64:
		fmt.Printf("%s: %d\n", query, current.Int())
	case reflect.Float32, reflect.Float64:
		fmt.Printf("%s: %f\n", query, current.Float())
	case reflect.String:
		fmt.Printf("%s: \"%s\"\n", query, current.String())
	case reflect.Bool:
		fmt.Printf("%s: %t\n", query, current.Bool())
	default:
		fmt.Printf("%s: %v\n", query, current.Interface())
	}
}

// -------------------------------------------------------------------
// INI File Management
// -------------------------------------------------------------------

// SaveINI saves the INI file to the specified path
func SaveINI(iniFile *ini.File, filePath string) error {
	if iniFile == nil {
		return fmt.Errorf("iniFile is not initialized")
	}
	return iniFile.SaveTo(filePath)
}

// GetIniMappingValue retrieves the value for a given section.key from the INI file
func GetIniMappingValue(iniFile *ini.File, query string) (string, error) {
	index := strings.Index(query, ".")
	if index == -1 {
		return "", fmt.Errorf("input string '%s' does not contain a valid section.key format", query)
	}
	sectionName := query[:index]
	keyName := query[index+1:]

	section, err := iniFile.GetSection(sectionName)
	if err != nil {
		return "", fmt.Errorf("error finding section '%s': %v", sectionName, err)
	}

	key, err := section.GetKey(keyName)
	if err != nil {
		return "", fmt.Errorf("error finding key '%s' in section '%s': %v", keyName, sectionName, err)
	}

	return key.Value(), nil
}

// PrintIniMappings prints all section.key mappings from the INI file
func PrintIniMappings(iniFile *ini.File) {
	for _, section := range iniFile.Sections() {
		sectionName := section.Name()

		for _, key := range section.Keys() {
			keyName := key.Name()
			value := key.Value()
			fmt.Printf("(%s, %s) = %s\n", sectionName, keyName, value)
		}
	}
}

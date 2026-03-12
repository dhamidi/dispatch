package dispatch

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/dhamidi/uritemplate"
)

// BindHelpers populates the fields of a struct with dynamically generated
// URL helper functions. Each exported field must be a func type and must
// have a `route:"<name>"` struct tag identifying the route to bind.
//
// Function signatures determine how arguments map to route parameters:
//
//   - Each function argument corresponds to a route parameter, matched by
//     position to the template's declared variables (path variables first,
//     then query variables, in declaration order).
//   - Supported argument types: string, int, int64, int32, uint, uint64,
//     uint32, float64, bool, and any type implementing fmt.Stringer.
//   - Arguments are converted to strings using strconv or fmt.Stringer
//     and passed as Params to Router.Path.
//   - The return type must be exactly `string` (the generated path) or
//     `(string, error)` for routes where expansion could fail.
//
// BindHelpers panics if:
//   - dest is not a pointer to a struct
//   - a `route:"..."` tag references a route name not registered in the router
//   - the number of function arguments doesn't match the route's template variables
//   - an argument type is not one of the supported types
//
// BindHelpers is intended to be called once during program startup.
//
// Example:
//
//	var urls URLs
//	router.BindHelpers(&urls)
//	path := urls.AdminPostsEdit(42)  // "/admin/posts/42/edit"
func (r *Router) BindHelpers(dest interface{}) {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Struct {
		panic("dispatch: BindHelpers requires a pointer to a struct")
	}
	structVal := destVal.Elem()
	structType := structVal.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		routeName, ok := field.Tag.Lookup("route")
		if !ok {
			continue
		}

		fieldVal := structVal.Field(i)
		if field.Type.Kind() != reflect.Func {
			panic(fmt.Sprintf("dispatch: field %s must be a func type", field.Name))
		}

		// Look up the route
		rr, ok := r.byName[routeName]
		if !ok {
			panic(fmt.Sprintf("dispatch: route %q not found (field %s)", routeName, field.Name))
		}

		// Get template variable names in declaration order
		varNames := orderedTemplateVarNames(rr.Template)

		// Validate argument count
		funcType := field.Type
		if funcType.NumIn() != len(varNames) {
			panic(fmt.Sprintf(
				"dispatch: field %s has %d args but route %q has %d template variables",
				field.Name, funcType.NumIn(), routeName, len(varNames),
			))
		}

		// Validate return type
		validateHelperReturnType(funcType, field.Name)

		// Validate argument types
		for j := 0; j < funcType.NumIn(); j++ {
			validateHelperArgType(funcType.In(j), field.Name, j)
		}

		// Create the function via reflect.MakeFunc
		fn := r.makeHelperFunc(routeName, varNames, funcType)
		fieldVal.Set(fn)
	}
}

// makeHelperFunc creates a reflect.Value containing a function that generates
// a URL path for the named route using positional arguments.
func (r *Router) makeHelperFunc(routeName string, varNames []string, funcType reflect.Type) reflect.Value {
	returnsError := funcType.NumOut() == 2

	return reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		params := make(Params, len(args))
		for i, arg := range args {
			params[varNames[i]] = helperArgToString(arg)
		}
		path, err := r.Path(routeName, params)
		if returnsError {
			var errVal reflect.Value
			if err != nil {
				errVal = reflect.ValueOf(&err).Elem()
			} else {
				errVal = reflect.Zero(funcType.Out(1))
			}
			return []reflect.Value{reflect.ValueOf(path), errVal}
		}
		if err != nil {
			panic(fmt.Sprintf("dispatch: URL generation failed for %q: %v", routeName, err))
		}
		return []reflect.Value{reflect.ValueOf(path)}
	})
}

var paramValueType = reflect.TypeOf((*ParamValue)(nil)).Elem()

// helperArgToString converts a reflect.Value to its string representation
// for use as a route parameter. ParamValue and fmt.Stringer implementations
// take precedence over the default numeric/bool formatting.
func helperArgToString(v reflect.Value) string {
	// ParamValue types know how to format themselves as URL params.
	if v.Type().Implements(paramValueType) {
		return v.Interface().(ParamValue).String()
	}
	// Check fmt.Stringer so named types with a String method
	// use their custom formatting rather than the underlying kind.
	if v.Type().Implements(stringerType) {
		return v.Interface().(fmt.Stringer).String()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float64, reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	default:
		panic(fmt.Sprintf("dispatch: unsupported argument type %s", v.Type()))
	}
}

var stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()

// validateHelperReturnType checks that a helper function's return type is
// either `string` or `(string, error)`.
func validateHelperReturnType(ft reflect.Type, fieldName string) {
	switch ft.NumOut() {
	case 1:
		if ft.Out(0).Kind() != reflect.String {
			panic(fmt.Sprintf("dispatch: field %s must return string or (string, error)", fieldName))
		}
	case 2:
		if ft.Out(0).Kind() != reflect.String {
			panic(fmt.Sprintf("dispatch: field %s first return value must be string", fieldName))
		}
		if !ft.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			panic(fmt.Sprintf("dispatch: field %s second return value must be error", fieldName))
		}
	default:
		panic(fmt.Sprintf("dispatch: field %s must return string or (string, error)", fieldName))
	}
}

// validateHelperArgType checks that a function argument type is one of the
// supported types for route parameter conversion.
func validateHelperArgType(t reflect.Type, fieldName string, argIdx int) {
	switch t.Kind() {
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		return
	default:
		if t.Implements(paramValueType) || t.Implements(stringerType) {
			return
		}
		panic(fmt.Sprintf(
			"dispatch: field %s arg %d has unsupported type %s",
			fieldName, argIdx, t,
		))
	}
}

// orderedTemplateVarNames returns the variable names from a URI template
// in their declaration order (path variables first, then query).
func orderedTemplateVarNames(t *uritemplate.Template) []string {
	raw := t.String()
	var names []string
	for i := 0; i < len(raw); i++ {
		if raw[i] == '{' {
			end := strings.IndexByte(raw[i:], '}')
			if end < 0 {
				break
			}
			body := raw[i+1 : i+end]
			// Strip URI template operators (+, #, ., /, ;, ?, &)
			if len(body) > 0 {
				first := body[0]
				if first == '+' || first == '#' || first == '.' || first == '/' || first == ';' || first == '?' || first == '&' {
					body = body[1:]
				}
			}
			// Handle comma-separated vars in expressions like {?q,page}
			for _, part := range strings.Split(body, ",") {
				name := strings.TrimRight(part, "*")
				if colonIdx := strings.IndexByte(name, ':'); colonIdx >= 0 {
					name = name[:colonIdx]
				}
				name = strings.TrimSpace(name)
				if name != "" {
					names = append(names, name)
				}
			}
			i += end
		}
	}
	return names
}

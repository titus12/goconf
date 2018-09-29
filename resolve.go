package goconf

import (
	"flag"
	"fmt"
	"reflect"
	"strings"

	"bitbucket.org/funplus/golib/str"
)

func HasArg(fs *flag.FlagSet, s string) bool {
	var found bool
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == s {
			found = true
		}
	})
	return found
}

func innserResolve(options interface{}, flagSet *flag.FlagSet, cfg map[string]interface{}, tomap map[string]interface{}, autoSet bool, Log func(string)) {
	val := reflect.ValueOf(options).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			var fieldPtr reflect.Value
			switch val.FieldByName(field.Name).Kind() {
			case reflect.Struct:
				fieldPtr = val.FieldByName(field.Name).Addr()
			case reflect.Ptr:
				fieldPtr = reflect.Indirect(val).FieldByName(field.Name)
			}
			if !fieldPtr.IsNil() {
				innserResolve(fieldPtr.Interface(), flagSet, cfg, tomap, autoSet, Log)
			}
			continue
		}

		var v interface{}
		flagName := field.Tag.Get("flag")
		cfgName := field.Tag.Get("cfg")
		defaultVal := field.Tag.Get("default")

		if flagName == "" {
			flagName = str.ToSnakeCase(field.Name)
		}

		if cfgName == "" {
			cfgName = strings.Replace(flagName, "-", "_", -1)
		}

		if autoSet {
			if flagSet.Lookup(flagName) == nil {
				if defaultVal != "" {
					v = defaultVal
				} else {
					v = val.Field(i).Interface()
				}
				if err := coerceAutoSet(v, val.FieldByName(field.Name).Interface(), flagSet, flagName); err != nil {
					Log(fmt.Sprintf("auto flag fail, name: %s val: %v err: %s", flagName, v, err.Error()))
				} else {
					Log(fmt.Sprintf("auto flag succ, name: %s val: %v", flagName, v))
				}
			}
		} else {
			// resolve the flags according to priority
			if flagSet != nil && HasArg(flagSet, flagName) { // command line flag value
				flagInst := flagSet.Lookup(flagName)
				v = flagInst.Value.String()
			} else if cfgVal, ok := cfg[cfgName]; ok { // config file value
				v = cfgVal
			} else if defaultVal != "" { // default value
				v = defaultVal
			} else {
				v = val.Field(i).Interface()
			}
			fieldVal := val.FieldByName(field.Name)
			coerced, err := coerce(v, fieldVal.Interface(), field.Tag.Get("arg"))
			if err != nil {
				Log(fmt.Sprintf("coerce fail: %v for %s (%+v) - %s", v, field.Name, fieldVal, err))
			}
			fieldVal.Set(reflect.ValueOf(coerced))
			if tomap != nil {
				if err == nil {
					tomap[flagName] = coerced
				} else {
					tomap[flagName] = v
				}
			}
		}
	}
}
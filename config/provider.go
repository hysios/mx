package config

type ConfigProvider interface {
	LookupPath(selector string) (val *Value, ok bool)
	Set(selector string, val interface{}) interface{}
	Update(vals map[string]interface{}) Map
	Data() Map
}

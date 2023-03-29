# MX is microservice gateway

## install mx framework cli
```

```
## Declare config 

### setup redis backend config
```shell
consul services register -name="mx.Config" -meta=service_type=config_provider -meta=targetURI=redis://127.0.0.1:6379/mx.config -address=127.0.0.1 -port=6379
```

### used cli config set

set a config key value
```shell
mx config set -key=key=value
```

get a config key value
```shell
mx config get -key=key
```

cat all config 
```shell
mx config cat
```

update all config 
```shell
mx config update --data="{"a": "b"}"
```

update config from a json file
```shell
mx config update --data=@/path/to/file.json
```



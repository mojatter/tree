module jsonschema

go 1.24.0

toolchain go1.24.4

require (
	github.com/google/jsonschema-go v0.4.2
	github.com/mojatter/tree v0.0.0
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
)

require go.yaml.in/yaml/v3 v3.0.4 // indirect

replace github.com/mojatter/tree => ../../

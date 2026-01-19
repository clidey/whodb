// This helps use auto generate libraries you can use in the language of
// your choice. You can have multiple generators if you use multiple languages.
// Just ensure that the output_dir is different for each generator.
generator target {
    // Valid values: "python/pydantic", "typescript", "ruby/sorbet", "rest/openapi"
    output_type "go"

    // Where the generated code will be saved (relative to baml_src/)
    output_dir "../"

    // The version of the BAML package you have installed (e.g. same version as your baml-py or @boundaryml/baml).
    // The BAML VSCode extension version should also match this version.
    version "0.217.0"

    // 'baml-cli generate' will run this after generating go code
    // This command will be run from within $output_dir/baml_client
    on_generate "gofmt -w . && goimports -w ."

    // Your Go packages name as specified in go.mod
    // We need this to generate correct imports in the generated baml_client
    client_package_name "github.com/clidey/whodb/core"
}

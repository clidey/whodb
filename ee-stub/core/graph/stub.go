package graph

import "github.com/99designs/gqlgen/graphql"

// Stub types for CE builds
// This allows go mod tidy to work in CE mode

type Config struct {
	Resolvers interface{}
}

type ExecutableSchema interface {
	graphql.ExecutableSchema
}

func NewExecutableSchema(config Config) ExecutableSchema {
	return nil
}

func NewResolver() interface{} {
	return nil
}

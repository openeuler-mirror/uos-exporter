package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var createBuilderTest = []struct {
	name        string
	pipe        bool
	compression bool
	metadata    bool
	want        Builder
}{
	{"default", false, false, false, &ArgumentBuilder{
		Compressed:   false,
		WithMetadata: false,
	}},
	{"compressed", false, true, false, &ArgumentBuilder{
		Compressed:   true,
		WithMetadata: false,
	}},
	{"include", false, false, true, &ArgumentBuilder{
		Compressed:   false,
		WithMetadata: true,
	}},
	{"compressedInclude", false, true, true, &ArgumentBuilder{
		Compressed:   true,
		WithMetadata: true,
	}},
	{"pipe", true, false, false, &PipeBuilder{}},
}


// TODO: implement

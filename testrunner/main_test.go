package testrunner

import (
	"os"
	"testing"

	"github.com/fatih/color"
)

func TestMain(m *testing.M) {
	color.NoColor = true

	os.Exit(m.Run())
}

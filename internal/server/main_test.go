package server

import (
	"os"
	"testing"

	"github.com/nodemode-ai/form.md/internal/agentlookup/testguard"
)

func TestMain(m *testing.M) { os.Exit(testguard.Run(m)) }

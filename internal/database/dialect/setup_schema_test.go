package dialect

import "testing"

func TestClassifySetupSchema(t *testing.T) {
	t.Parallel()

	full := map[string]struct{}{
		"account_policies":  {},
		"auth_sessions":     {},
		"credentials":       {},
		"install_state":     {},
		"schema_migrations": {},
		"users":             {},
	}
	withoutInstallState := make(map[string]struct{}, len(full))
	for name := range full {
		if name != "install_state" {
			withoutInstallState[name] = struct{}{}
		}
	}

	tests := []struct {
		name   string
		tables map[string]struct{}
		want   SetupSchema
	}{
		{name: "empty", tables: nil, want: SetupSchemaEmpty},
		{name: "ledger only", tables: map[string]struct{}{"schema_migrations": {}}, want: SetupSchemaLedgerOnly},
		{name: "application schema", tables: full, want: SetupSchemaApplication},
		{name: "application schema without install state", tables: withoutInstallState, want: SetupSchemaForeign},
		{name: "application table without ledger", tables: map[string]struct{}{"users": {}}, want: SetupSchemaForeign},
		{name: "unknown table", tables: map[string]struct{}{"schema_migrations": {}, "unrelated": {}}, want: SetupSchemaForeign},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifySetupSchema(test.tables); got != test.want {
				t.Fatalf("ClassifySetupSchema() = %d, want %d", got, test.want)
			}
		})
	}
}

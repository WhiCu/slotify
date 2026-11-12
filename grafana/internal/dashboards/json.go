package dashboards

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
)

func RenderJSON(b *dashboard.DashboardBuilder, name string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("dashboard name cannot be empty")
	}

	model, err := b.Build()
	if err != nil {
		return nil, fmt.Errorf("build dashboard: %w", err)
	}

	if model.Uid == nil {
		model.Uid = &name
	}

	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal dashboard: %w", err)
	}

	return data, nil
}

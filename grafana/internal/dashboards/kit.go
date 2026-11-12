package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
)

func metricDatasource() common.DataSourceRef {
	dsType := "victoriametrics-metrics-datasource"
	dsUID := "victoriametrics"
	return common.DataSourceRef{Type: &dsType, Uid: &dsUID}
}

func newBaseDashboard(title string, tags ...string) *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder(title).
		Tags(tags).
		Refresh("1m").
		Time("now-6h", "now").
		Tooltip(dashboard.DashboardCursorSyncCrosshair)
}

func promQuery(expr, legend, refID string) *prometheus.DataqueryBuilder {
	q := prometheus.NewDataqueryBuilder().
		Expr(expr).
		RefId(refID)
	if legend != "" {
		q = q.LegendFormat(legend)
	}
	return q
}

func reduceLast() *common.ReduceDataOptionsBuilder {
	return common.NewReduceDataOptionsBuilder().
		Values(false).
		Calcs([]string{"last"})
}

type thresholdStep struct {
	value float64
	color string
}

func step(value float64, color string) thresholdStep {
	return thresholdStep{value: value, color: color}
}

func thresholds(baseColor string, steps ...thresholdStep) *dashboard.ThresholdsConfigBuilder {
	ts := []dashboard.Threshold{{Color: baseColor}}
	for _, s := range steps {
		value := s.value
		ts = append(ts, dashboard.Threshold{Value: &value, Color: s.color})
	}
	return dashboard.NewThresholdsConfigBuilder().
		Mode(dashboard.ThresholdsModeAbsolute).
		Steps(ts)
}

func upDownMappings() []dashboard.ValueMapping {
	downText, downColor := "DOWN", "red"
	upText, upColor := "UP", "green"
	downIdx, upIdx := int32(0), int32(1)
	return []dashboard.ValueMapping{
		{
			ValueMap: &dashboard.ValueMap{
				Type: dashboard.MappingTypeValueToText,
				Options: map[string]dashboard.ValueMappingResult{
					"0": {Text: &downText, Color: &downColor, Index: &downIdx},
					"1": {Text: &upText, Color: &upColor, Index: &upIdx},
				},
			},
		},
	}
}

func hiddenLegend() *common.VizLegendOptionsBuilder {
	return common.NewVizLegendOptionsBuilder().
		DisplayMode(common.LegendDisplayModeHidden)
}

func defaultLegend() *common.VizLegendOptionsBuilder {
	return common.NewVizLegendOptionsBuilder().
		DisplayMode(common.LegendDisplayModeList).
		Placement(common.LegendPlacementBottom).
		ShowLegend(true)
}

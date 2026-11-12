package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/bargauge"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/gauge"
	"github.com/grafana/grafana-foundation-sdk/go/heatmap"
	"github.com/grafana/grafana-foundation-sdk/go/piechart"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/statetimeline"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

func BuildAppDashboard() ([]byte, error) {
	db := newBaseDashboard("Slotify - Application Metrics", "slotify", "red", "use", "golden-signals").
		Uid("slotify-app").
		WithRow(dashboard.NewRowBuilder("Golden Signals & SLO (окна 24h)")).
		WithPanel(availabilitySLOGauge()).
		WithPanel(latencySLOGauge()).
		WithPanel(dbUpStat()).
		WithRow(dashboard.NewRowBuilder("Health Probe /healthz (OTel httpcheck) - SLA пробера")).
		WithPanel(probeSLAStat()).
		WithPanel(probeTimelinePanel()).
		WithRow(dashboard.NewRowBuilder("Application Traffic & Errors (RED)")).
		WithPanel(appRPSPanel()).
		WithPanel(appStatusCodesPanel()).
		WithPanel(appErrorRatioPanel()).
		WithPanel(appLatencyPanel()).
		WithPanel(latencyHeatmapPanel()).
		WithRow(dashboard.NewRowBuilder("Runtime & Resources (USE)")).
		WithPanel(runtimeMemoryPanel()).
		WithPanel(runtimeGoroutinesPanel())

	return RenderJSON(db, "slotify-app")
}

func availabilitySLOGauge() *gauge.PanelBuilder {
	return gauge.NewPanelBuilder().
		Title("API Availability (SLI-1)").
		Description("Golden Signals / Errors: доля запросов без 5xx за 24 ч. SLO 99.5% (бюджет ошибок 0.5%); ниже 99% - жёлтая зона, ниже 99.5% - SLO нарушен.").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 8}).
		WithTarget(promQuery(
			`sum(rate(ogen_server_request_count_total{http_response_status_code!~"5.."}[$__rate_interval])) / sum(rate(ogen_server_request_count_total[$__rate_interval]))`,
			"Availability", "A",
		)).
		Unit("percentunit").
		Min(0).
		Max(1).
		Orientation(common.VizOrientationAuto).
		ShowThresholdMarkers(true).
		ShowThresholdLabels(true).
		ReduceOptions(reduceLast()).
		Thresholds(thresholds("red", step(0.99, "yellow"), step(0.995, "green")))
}

func latencySLOGauge() *gauge.PanelBuilder {
	return gauge.NewPanelBuilder().
		Title("Latency SLO (<= 750 ms)").
		Description("Golden Signals / Latency: доля запросов, обработанных быстрее 750 мс, за 24 ч. SLO 95% - цель из отчёта; красный - цель не выполняется.").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 8}).
		WithTarget(promQuery(
			`sum(rate(ogen_server_duration_milliseconds_bucket{le="750"}[$__rate_interval])) / sum(rate(ogen_server_duration_milliseconds_count[$__rate_interval]))`,
			"Fast requests", "A",
		)).
		Unit("percentunit").
		Min(0).
		Max(1).
		Orientation(common.VizOrientationAuto).
		ShowThresholdMarkers(true).
		ShowThresholdLabels(true).
		ReduceOptions(reduceLast()).
		Thresholds(thresholds("red", step(0.95, "green")))
}

func dbUpStat() *stat.PanelBuilder {
	return stat.NewPanelBuilder().
		Title("PostgreSQL Up").
		Description("Golden Signals / Saturation: жива ли база (pg_up из postgres-экспортера). SLO хранилища: доступность >= 99.5% времени. No data = экспортер молчит.").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 8}).
		WithTarget(promQuery(`max(pg_up)`, "Postgres", "A")).
		GraphMode(common.BigValueGraphModeNone).
		ColorMode(common.BigValueColorModeBackgroundSolid).
		TextMode(common.BigValueTextModeValue).
		Orientation(common.VizOrientationAuto).
		NoValue("no data").
		ReduceOptions(reduceLast()).
		Mappings(upDownMappings()).
		Thresholds(thresholds("red", step(1, "green")))
}

func probeSLAStat() *stat.PanelBuilder {
	return stat.NewPanelBuilder().
		Title("Probe SLA /healthz ($__rate_interval)").
		Description("Подсчёт SLA пробера: доля успешных проверок /healthz за 24 ч = сумма значений точек класса 2xx / общее число точек (одна точка на каждую проверку, включая сетевые провалы без http.status_code).").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 6}).
		WithTarget(promQuery(
			`sum(sum_over_time({__name__="httpcheck.status", http.status_class="2xx"}[$__rate_interval])) / sum(count_over_time({__name__="httpcheck.status", http.status_class="2xx"}[$__rate_interval]))`,
			"Probe SLA", "A",
		)).
		Unit("percentunit").
		GraphMode(common.BigValueGraphModeNone).
		ColorMode(common.BigValueColorModeValue).
		Orientation(common.VizOrientationAuto).
		ReduceOptions(reduceLast()).
		Thresholds(thresholds("red", step(0.9, "yellow"), step(0.995, "green")))
}

func probeTimelinePanel() *statetimeline.PanelBuilder {
	return statetimeline.NewPanelBuilder().
		Title("Probe Status Timeline").
		Description("Состояние пробы /healthz поминутно (max_over_time по классу 2xx): зелёный - был 2xx-ответ, красный - за минуту не было ни одного. Видны моменты падений, которые портят Probe SLA слева.").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 18}).
		WithTarget(promQuery(
			`max(max_over_time({__name__="httpcheck.status", http.status_class="2xx"}[1m]))`,
			"/healthz", "A",
		)).
		Mappings(upDownMappings()).
		Thresholds(thresholds("red", step(1, "green"))).
		MergeValues(true).
		RowHeight(0.7).
		Legend(hiddenLegend())
}

func appRPSPanel() *timeseries.PanelBuilder {
	return timeseries.NewPanelBuilder().
		Title("Request Rate (RPS)").
		Description("RED: Rate - суммарный входящий поток запросов.").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 12}).
		WithTarget(promQuery(`sum(rate(ogen_server_request_count_total[$__rate_interval]))`, "Total RPS", "A")).
		Unit("reqps").
		Min(0).
		Legend(defaultLegend())
}

func appStatusCodesPanel() *piechart.PanelBuilder {
	return piechart.NewPanelBuilder().
		Title("Responses by HTTP Status").
		Description("RED: доля каждого статус-кода за выбранный период (increase по $__range). Хорошо видно, сколько ответов уходит в 4xx/5xx.").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 12}).
		WithTarget(promQuery(
			`sum by (http_response_status_code) (increase(ogen_server_request_count_total[$__range]))`,
			"HTTP {{http_response_status_code}}", "A",
		)).
		PieType(piechart.PieChartTypeDonut).
		DisplayLabels([]piechart.PieChartLabels{
			piechart.PieChartLabelsName,
			piechart.PieChartLabelsPercent,
		}).
		ReduceOptions(reduceLast()).
		Orientation(common.VizOrientationAuto).
		Legend(piechart.NewPieChartLegendOptionsBuilder().
			DisplayMode(common.LegendDisplayModeList).
			Placement(common.LegendPlacementRight).
			ShowLegend(true).
			Values([]piechart.PieChartLegendValues{
				piechart.PieChartLegendValuesValue,
				piechart.PieChartLegendValuesPercent,
			}))
}

func appErrorRatioPanel() *timeseries.PanelBuilder {
	return timeseries.NewPanelBuilder().
		Title("5xx Error Ratio").
		Description("RED: Errors - доля 5xx ошибок; красная линия - порог алерта slotify-backend-error-rate (5%).").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 12}).
		WithTarget(promQuery(
			`sum(rate(ogen_server_request_count_total{http_response_status_code=~"5.."}[$__rate_interval])) / clamp_min(sum(rate(ogen_server_request_count_total[$__rate_interval])), 1e-9)`,
			"5xx Ratio", "A",
		)).
		Unit("percentunit").
		Min(0).
		Thresholds(thresholds("green", step(0.05, "red"))).
		Legend(defaultLegend())
}

func appLatencyPanel() *timeseries.PanelBuilder {
	return timeseries.NewPanelBuilder().
		Title("Request Duration").
		Description("RED: Duration - квантили времени обработки запросов (p50/p95/p99).").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 12}).
		WithTarget(promQuery(`histogram_quantile(0.50, sum by (le) (rate(ogen_server_duration_milliseconds_bucket[$__rate_interval])))`, "p50", "A")).
		WithTarget(promQuery(`histogram_quantile(0.95, sum by (le) (rate(ogen_server_duration_milliseconds_bucket[$__rate_interval])))`, "p95", "B")).
		WithTarget(promQuery(`histogram_quantile(0.99, sum by (le) (rate(ogen_server_duration_milliseconds_bucket[$__rate_interval])))`, "p99", "C")).
		Unit("ms").
		Min(0).
		Legend(defaultLegend())
}

func latencyHeatmapPanel() *heatmap.PanelBuilder {
	return heatmap.NewPanelBuilder().
		Title("Latency Distribution (Heatmap)").
		Description("Тепловая карта задержек по бакетам гистограммы (le): где концентрируются запросы и каков хвост. Ось Y - границы бакетов в мс.").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 24}).
		WithTarget(promQuery(
			`sum by (le) (increase(ogen_server_duration_milliseconds_bucket[$__rate_interval]))`,
			"", "A",
		).Format(prometheus.PromQueryFormatHeatmap)).
		Unit("ms").
		YAxis(heatmap.NewYAxisConfigBuilder().Unit("ms")).
		ShowLegend()
}

func runtimeMemoryPanel() *timeseries.PanelBuilder {
	return timeseries.NewPanelBuilder().
		Title("Go Runtime Memory").
		Description("USE: Utilization - память, используемая рантаймом Go, по подам (лейбл host_name).").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 12}).
		WithTarget(promQuery(`sum by (host_name) (go_memory_used_bytes{service_name="slotify"})`, "{{host_name}}", "A")).
		Unit("bytes").
		Min(0).
		Legend(defaultLegend())
}

func runtimeGoroutinesPanel() *bargauge.PanelBuilder {
	return bargauge.NewPanelBuilder().
		Title("Active Goroutines").
		Description("USE: Saturation - число активных горутин по подам; красная зона - порог алерта slotify-backend-goroutine-leak (500). Норма - десятки.").
		Datasource(metricDatasource()).
		GridPos(dashboard.GridPos{H: 8, W: 24}).
		WithTarget(promQuery(`max by (host_name) (go_goroutine_count{service_name="slotify"})`, "{{host_name}}", "A")).
		Unit("short").
		Min(0).
		DisplayMode(common.BarGaugeDisplayModeGradient).
		Orientation(common.VizOrientationHorizontal).
		Sizing(common.BarGaugeSizingAuto).
		ReduceOptions(reduceLast()).
		Thresholds(thresholds("green", step(500, "red")))
}

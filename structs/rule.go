package structs

// Rule struct
type Rule struct {
	Name            string
	Backend         string `hcl:"backend"`
	BackendInstance Backender
	Comparison      string  `hcl:"comparison"`
	ComparisonValue float64 `hcl:"comparison_value,float"`
	Action          string  `hcl:"action"`
	ActionValue     int     `hcl:"action_value,int"`
	MetricName      string  `hcl:"metric_name"`
	MetricNamespace string  `hcl:"metric_namespace"`
	DimensionName   string  `hcl:"dimension_name"`
	DimensionValue  string  `hcl:"dimension_value"`
	Period          string  `hcl:"cron"`
}

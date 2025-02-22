package clickhouse

import (
	"context"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/highlight-run/highlight/backend/model"
	"github.com/openlyinc/pointy"

	modelInputs "github.com/highlight-run/highlight/backend/private-graph/graph/model"
)

const MetricNamesTable = "trace_metrics"

var metricsTableConfig = model.TableConfig{
	AttributesColumn: TracesTableNoDefaultConfig.AttributesColumn,
	BodyColumn:       TracesTableNoDefaultConfig.BodyColumn,
	MetricColumn:     ptr.String("MetricValue"),
	KeysToColumns:    TracesTableNoDefaultConfig.KeysToColumns,
	ReservedKeys:     TracesTableNoDefaultConfig.ReservedKeys,
	SelectColumns:    TracesTableNoDefaultConfig.SelectColumns,
	TableName:        TracesTableNoDefaultConfig.TableName,
}

var metricsSamplingTableConfig = model.TableConfig{
	AttributesColumn: metricsTableConfig.AttributesColumn,
	BodyColumn:       metricsTableConfig.BodyColumn,
	MetricColumn:     metricsTableConfig.MetricColumn,
	KeysToColumns:    metricsTableConfig.KeysToColumns,
	ReservedKeys:     metricsTableConfig.ReservedKeys,
	SelectColumns:    metricsTableConfig.SelectColumns,
	TableName:        TracesSamplingTable,
}

var MetricsSampleableTableConfig = SampleableTableConfig{
	tableConfig:         metricsTableConfig,
	samplingTableConfig: metricsSamplingTableConfig,
	sampleSizeRows:      20_000_000,
}

func (client *Client) ReadEventMetrics(ctx context.Context, projectID int, params modelInputs.QueryInput, column *string, metricTypes []modelInputs.MetricAggregator, groupBy []string, nBuckets *int, bucketBy string, bucketWindow *int, limit *int, limitAggregator *modelInputs.MetricAggregator, limitColumn *string) (*modelInputs.MetricsBuckets, error) {
	columnDeref := ""
	if column != nil {
		columnDeref = *column
	}

	expressions := []*modelInputs.MetricExpressionInput{}
	for _, t := range metricTypes {
		expressions = append(expressions, &modelInputs.MetricExpressionInput{
			Aggregator: t,
			Column:     columnDeref,
		})
	}

	params.Query = params.Query + " " + modelInputs.ReservedTraceKeyMetricName.String() + "=" + columnDeref
	return client.ReadMetrics(ctx, ReadMetricsInput{
		SampleableConfig: MetricsSampleableTableConfig,
		ProjectIDs:       []int{projectID},
		Params:           params,
		GroupBy:          groupBy,
		BucketCount:      nBuckets,
		BucketWindow:     bucketWindow,
		BucketBy:         bucketBy,
		Limit:            limit,
		LimitAggregator:  limitAggregator,
		LimitColumn:      limitColumn,
		Expressions:      expressions,
	})
}

func (client *Client) ReadWorkspaceMetricCounts(ctx context.Context, projectIDs []int, params modelInputs.QueryInput) (*modelInputs.MetricsBuckets, error) {
	params.Query = params.Query + " " + modelInputs.ReservedTraceKeyMetricValue.String() + " exists"
	// 12 buckets - 12 months in a year, or 12 weeks in a quarter
	return client.ReadMetrics(ctx, ReadMetricsInput{
		SampleableConfig: MetricsSampleableTableConfig,
		ProjectIDs:       projectIDs,
		Params:           params,
		BucketCount:      pointy.Int(12),
		BucketBy:         modelInputs.MetricBucketByTimestamp.String(),
		Expressions: []*modelInputs.MetricExpressionInput{{
			Aggregator: modelInputs.MetricAggregatorCount,
		}},
	})
}

func (client *Client) MetricsKeys(ctx context.Context, projectID int, startDate time.Time, endDate time.Time, query *string, typeArg *modelInputs.KeyType) ([]*modelInputs.QueryKey, error) {
	if typeArg != nil && *typeArg == modelInputs.KeyTypeNumeric {
		metricKeys, err := KeysAggregated(ctx, client, MetricNamesTable, projectID, startDate, endDate, query, typeArg, nil)
		if err != nil {
			return nil, err
		}

		return metricKeys, nil
	}

	return client.TracesKeys(ctx, projectID, startDate, endDate, query, typeArg)
}

func (client *Client) MetricsKeyValues(ctx context.Context, projectID int, keyName string, startDate time.Time, endDate time.Time, query *string, limit *int) ([]string, error) {
	return KeyValuesAggregated(ctx, client, TraceKeyValuesTable, projectID, keyName, startDate, endDate, query, limit, nil)
}

func (client *Client) MetricsLogLines(ctx context.Context, projectID int, params modelInputs.QueryInput) ([]*modelInputs.LogLine, error) {
	return logLines(ctx, client, metricsTableConfig, projectID, params)
}

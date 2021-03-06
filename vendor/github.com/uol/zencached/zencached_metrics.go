package zencached

//
// A plugable interface to collect metrics.
// @author rnojiri
//

const (
	metricNodeDistribution      string = "zencached.node.distribution.count"
	metricNodeConnAvailableTime string = "zencached.node.conn.available.time"
	metricOperationCount        string = "zencached.operation.count"
	metricOperationTime         string = "zencached.operation.time"
	metricCacheMiss             string = "zencached.cache.miss"
	metricCacheHit              string = "zencached.cache.hit"
	tagNodeName                 string = "node"
	tagOperationName            string = "operation"
)

// MetricsCollector - the interface
type MetricsCollector interface {

	// Count - collects a metric from zencached (tags are a list of alternated key and values)
	Count(value float64, metric string, tags ...interface{})

	// Maximum - collects a metric from zencached (tags are a list of alternated key and values)
	Maximum(value float64, metric string, tags ...interface{})
}

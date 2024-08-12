package telemetry

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"go.opentelemetry.io/otel"
)

var meter = otel.Meter("go.perf_stats")
var cpuGauge, _ = meter.Float64Gauge("cpu_usage")
var memoryGauge, _ = meter.Int64Gauge("allocated_mb")
var gcGauge, _ = meter.Int64Gauge("gc_cycles_per_second")
var pauseTotalGauge, _ = meter.Int64Gauge("stop_the_world_nanosecs")

func InstrumentPerfStats(ctx context.Context) {
	go func() {
		var memStats runtime.MemStats
		ticker := time.NewTicker(time.Second * 30)

		for {
			select {
			case <-ticker.C:
				runtime.ReadMemStats(&memStats)

				cpuUsage, err := cpu.Percent(time.Minute, false)
				if err == nil {
					cpuGauge.Record(ctx, cpuUsage[0])
				} else {
					fmt.Println("failed to read cpu usage", err)
				}

				memoryGauge.Record(ctx, int64(memStats.Alloc/1_000_000))
				gcGauge.Record(ctx, int64(memStats.NumGC))
				pauseTotalGauge.Record(ctx, int64(memStats.PauseTotalNs))
			case <-ctx.Done():
				return
			}
		}
	}()
}

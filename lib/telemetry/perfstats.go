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
var liveObjectsGauge, _ = meter.Int64Gauge("live_objects")
var goroutineGauge, _ = meter.Int64Gauge("goroutine_count")

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
				liveObjectsGauge.Record(ctx, int64(memStats.Mallocs)-int64(memStats.Frees))
				goroutineGauge.Record(ctx, int64(runtime.NumGoroutine()))
			case <-ctx.Done():
				return
			}
		}
	}()
}

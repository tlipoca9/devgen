package generator_test

import (
	"encoding/json"
	"time"

	"github.com/onsi/gomega/gmeasure"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tlipoca9/devgen/cmd/enumgen/generator"
)

// Benchmark tests for generated enum methods using Ginkgo's gmeasure
// Run with: go test -v ./cmd/enumgen/generator/... -count=1

var _ = Describe("Benchmark", func() {
	var experiment *gmeasure.Experiment

	BeforeEach(func() {
		experiment = gmeasure.NewExperiment(CurrentSpecReport().LeafNodeText)
		AddReportEntry(experiment.Name, experiment)
	})

	Describe("IsValid", func() {
		It("benchmarks IsValid with valid value", func() {
			experiment.SampleDuration("IsValid/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionString.IsValid()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("IsValid/valid")
			// Mean should be less than 10µs for 1000 iterations
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks IsValid with invalid value", func() {
			invalid := generator.GenerateOption(999)
			experiment.SampleDuration("IsValid/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = invalid.IsValid()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("IsValid/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("String", func() {
		It("benchmarks String with valid value", func() {
			experiment.SampleDuration("String/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionString.String()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("String/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks String with invalid value", func() {
			invalid := generator.GenerateOption(999)
			experiment.SampleDuration("String/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = invalid.String()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("String/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})
	})

	Describe("MarshalJSON", func() {
		It("benchmarks MarshalJSON direct call", func() {
			experiment.SampleDuration("MarshalJSON/direct", func(_ int) {
				for i := 0; i < 1000; i++ {
					_, _ = generator.GenerateOptionString.MarshalJSON()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("MarshalJSON/direct")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks MarshalJSON via json.Marshal", func() {
			experiment.SampleDuration("MarshalJSON/via_json_Marshal", func(_ int) {
				for i := 0; i < 1000; i++ {
					_, _ = json.Marshal(generator.GenerateOptionJSON)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("MarshalJSON/via_json_Marshal")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 200*time.Microsecond))
		})
	})

	Describe("UnmarshalJSON", func() {
		It("benchmarks UnmarshalJSON direct call", func() {
			data := []byte(`"text"`)
			var v generator.GenerateOption
			experiment.SampleDuration("UnmarshalJSON/direct", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = v.UnmarshalJSON(data)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("UnmarshalJSON/direct")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 200*time.Microsecond))
		})

		It("benchmarks UnmarshalJSON via json.Unmarshal", func() {
			data := []byte(`"sql"`)
			var v generator.GenerateOption
			experiment.SampleDuration("UnmarshalJSON/via_json_Unmarshal", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = json.Unmarshal(data, &v)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("UnmarshalJSON/via_json_Unmarshal")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 300*time.Microsecond))
		})
	})

	Describe("MarshalText", func() {
		It("benchmarks MarshalText", func() {
			experiment.SampleDuration("MarshalText", func(_ int) {
				for i := 0; i < 1000; i++ {
					_, _ = generator.GenerateOptionSQL.MarshalText()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("MarshalText")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 50*time.Microsecond))
		})
	})

	Describe("UnmarshalText", func() {
		It("benchmarks UnmarshalText", func() {
			data := []byte("sql")
			var v generator.GenerateOption
			experiment.SampleDuration("UnmarshalText", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = v.UnmarshalText(data)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("UnmarshalText")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 50*time.Microsecond))
		})
	})

	Describe("Value (SQL)", func() {
		It("benchmarks Value", func() {
			experiment.SampleDuration("Value", func(_ int) {
				for i := 0; i < 1000; i++ {
					_, _ = generator.GenerateOptionString.Value()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Value")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 50*time.Microsecond))
		})
	})

	Describe("Scan (SQL)", func() {
		It("benchmarks Scan with string", func() {
			var v generator.GenerateOption
			experiment.SampleDuration("Scan/string", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = v.Scan("json")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Scan/string")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})

		It("benchmarks Scan with bytes", func() {
			data := []byte("text")
			var v generator.GenerateOption
			experiment.SampleDuration("Scan/bytes", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = v.Scan(data)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Scan/bytes")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 50*time.Microsecond))
		})

		It("benchmarks Scan with nil", func() {
			var v generator.GenerateOption
			experiment.SampleDuration("Scan/nil", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = v.Scan(nil)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Scan/nil")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("Parse", func() {
		It("benchmarks Parse with valid name", func() {
			experiment.SampleDuration("Parse/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_, _ = generator.GenerateOptionEnums.Parse("string")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Parse/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})

		It("benchmarks Parse with invalid name", func() {
			experiment.SampleDuration("Parse/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_, _ = generator.GenerateOptionEnums.Parse("invalid")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Parse/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 200*time.Microsecond))
		})
	})

	Describe("Contains", func() {
		It("benchmarks Contains with valid value", func() {
			experiment.SampleDuration("Contains/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionEnums.Contains(generator.GenerateOptionJSON)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Contains/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks Contains with invalid value", func() {
			invalid := generator.GenerateOption(999)
			experiment.SampleDuration("Contains/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionEnums.Contains(invalid)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Contains/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("ContainsName", func() {
		It("benchmarks ContainsName with valid name", func() {
			experiment.SampleDuration("ContainsName/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionEnums.ContainsName("text")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("ContainsName/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})

		It("benchmarks ContainsName with invalid name", func() {
			experiment.SampleDuration("ContainsName/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionEnums.ContainsName("invalid")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("ContainsName/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})
	})

	Describe("Name", func() {
		It("benchmarks Name with valid value", func() {
			experiment.SampleDuration("Name/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionEnums.Name(generator.GenerateOptionSQL)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Name/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks Name with invalid value", func() {
			invalid := generator.GenerateOption(999)
			experiment.SampleDuration("Name/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionEnums.Name(invalid)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Name/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})
	})

	Describe("List", func() {
		It("benchmarks List", func() {
			experiment.SampleDuration("List", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionEnums.List()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("List")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("Names", func() {
		It("benchmarks Names", func() {
			experiment.SampleDuration("Names", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = generator.GenerateOptionEnums.Names()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("Names")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 50*time.Microsecond))
		})
	})
})

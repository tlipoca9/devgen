package generator_test

import (
	"encoding/csv"
	"encoding/json"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/onsi/gomega/gmeasure"
	"gopkg.in/yaml.v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Benchmark tests for individual validation annotations using Ginkgo's gmeasure
// Run with: go test -v ./cmd/validategen/generator/... -count=1

// Pre-compiled regex patterns (same as generated code)
var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	urlRegex      = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	uuidRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	alphaRegex    = regexp.MustCompile(`^[a-zA-Z]+$`)
	alphanumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	numericRegex  = regexp.MustCompile(`^[0-9]+$`)
	customRegex   = regexp.MustCompile(`^[A-Z]{2}-[0-9]{4}$`)
)

var _ = Describe("Benchmark", func() {
	var experiment *gmeasure.Experiment

	BeforeEach(func() {
		experiment = gmeasure.NewExperiment(CurrentSpecReport().LeafNodeText)
		AddReportEntry(experiment.Name, experiment)
	})

	Describe("@required", func() {
		It("benchmarks required string (valid)", func() {
			s := "hello"
			experiment.SampleDuration("required/string_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = s != ""
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("required/string_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks required string (invalid)", func() {
			s := ""
			experiment.SampleDuration("required/string_invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = s != ""
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("required/string_invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks required int (valid)", func() {
			n := 42
			experiment.SampleDuration("required/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n != 0
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("required/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks required slice (valid)", func() {
			s := []string{"a", "b"}
			experiment.SampleDuration("required/slice_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = len(s) > 0
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("required/slice_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks required pointer (valid)", func() {
			v := 42
			p := &v
			experiment.SampleDuration("required/pointer_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = p != nil
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("required/pointer_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("@min/@max", func() {
		It("benchmarks min int (valid)", func() {
			n := 10
			experiment.SampleDuration("min/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n >= 5
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("min/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks max int (valid)", func() {
			n := 10
			experiment.SampleDuration("max/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n <= 100
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("max/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks min string length (valid)", func() {
			s := "hello world"
			experiment.SampleDuration("min/string_len_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = len(s) >= 5
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("min/string_len_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks max string length (valid)", func() {
			s := "hello"
			experiment.SampleDuration("max/string_len_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = len(s) <= 100
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("max/string_len_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks min slice length (valid)", func() {
			s := []int{1, 2, 3, 4, 5}
			experiment.SampleDuration("min/slice_len_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = len(s) >= 3
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("min/slice_len_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("@len", func() {
		It("benchmarks len string (valid)", func() {
			s := "hello"
			experiment.SampleDuration("len/string_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = len(s) == 5
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("len/string_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks len slice (valid)", func() {
			s := []int{1, 2, 3}
			experiment.SampleDuration("len/slice_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = len(s) == 3
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("len/slice_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("@gt/@gte/@lt/@lte", func() {
		It("benchmarks gt int (valid)", func() {
			n := 10
			experiment.SampleDuration("gt/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n > 5
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("gt/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks gte int (valid)", func() {
			n := 10
			experiment.SampleDuration("gte/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n >= 10
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("gte/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks lt int (valid)", func() {
			n := 10
			experiment.SampleDuration("lt/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n < 20
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("lt/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks lte int (valid)", func() {
			n := 10
			experiment.SampleDuration("lte/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n <= 10
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("lte/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("@eq/@ne", func() {
		It("benchmarks eq string (valid)", func() {
			s := "active"
			experiment.SampleDuration("eq/string_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = s == "active"
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("eq/string_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks ne string (valid)", func() {
			s := "active"
			experiment.SampleDuration("ne/string_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = s != "deleted"
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("ne/string_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks eq int (valid)", func() {
			n := 1
			experiment.SampleDuration("eq/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n == 1
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("eq/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks eq bool (valid)", func() {
			b := true
			experiment.SampleDuration("eq/bool_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = b == true
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("eq/bool_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("@oneof", func() {
		It("benchmarks oneof string (valid, first)", func() {
			s := "debug"
			experiment.SampleDuration("oneof/string_first", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = s == "debug" || s == "info" || s == "warn" || s == "error"
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("oneof/string_first")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks oneof string (valid, last)", func() {
			s := "error"
			experiment.SampleDuration("oneof/string_last", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = s == "debug" || s == "info" || s == "warn" || s == "error"
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("oneof/string_last")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks oneof string (invalid)", func() {
			s := "invalid"
			experiment.SampleDuration("oneof/string_invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = s == "debug" || s == "info" || s == "warn" || s == "error"
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("oneof/string_invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})

		It("benchmarks oneof int (valid)", func() {
			n := 2
			experiment.SampleDuration("oneof/int_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = n == 1 || n == 2 || n == 3
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("oneof/int_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 10*time.Microsecond))
		})
	})

	Describe("@email", func() {
		It("benchmarks email (valid)", func() {
			s := "user@example.com"
			experiment.SampleDuration("email/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = emailRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("email/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 500*time.Microsecond))
		})

		It("benchmarks email (invalid)", func() {
			s := "invalid-email"
			experiment.SampleDuration("email/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = emailRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("email/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 200*time.Microsecond))
		})
	})

	Describe("@url", func() {
		It("benchmarks url (valid)", func() {
			s := "https://example.com/path?query=1"
			experiment.SampleDuration("url/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = urlRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("url/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 500*time.Microsecond))
		})

		It("benchmarks url (invalid)", func() {
			s := "not-a-url"
			experiment.SampleDuration("url/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = urlRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("url/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})
	})

	Describe("@uuid", func() {
		It("benchmarks uuid (valid)", func() {
			s := "550e8400-e29b-41d4-a716-446655440000"
			experiment.SampleDuration("uuid/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = uuidRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("uuid/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 300*time.Microsecond))
		})

		It("benchmarks uuid (invalid)", func() {
			s := "not-a-uuid"
			experiment.SampleDuration("uuid/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = uuidRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("uuid/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})
	})

	Describe("@ip/@ipv4/@ipv6", func() {
		It("benchmarks ip (valid ipv4)", func() {
			s := "192.168.1.1"
			experiment.SampleDuration("ip/ipv4_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = net.ParseIP(s) != nil
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("ip/ipv4_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 200*time.Microsecond))
		})

		It("benchmarks ip (valid ipv6)", func() {
			s := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
			experiment.SampleDuration("ip/ipv6_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = net.ParseIP(s) != nil
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("ip/ipv6_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 300*time.Microsecond))
		})

		It("benchmarks ip (invalid)", func() {
			s := "not-an-ip"
			experiment.SampleDuration("ip/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = net.ParseIP(s) != nil
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("ip/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 200*time.Microsecond))
		})

		It("benchmarks ipv4 specific", func() {
			s := "192.168.1.1"
			experiment.SampleDuration("ipv4/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					ip := net.ParseIP(s)
					_ = ip != nil && ip.To4() != nil
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("ipv4/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 200*time.Microsecond))
		})

		It("benchmarks ipv6 specific", func() {
			s := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
			experiment.SampleDuration("ipv6/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					ip := net.ParseIP(s)
					_ = ip != nil && ip.To4() == nil
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("ipv6/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 300*time.Microsecond))
		})
	})

	Describe("@alpha/@alphanum/@numeric", func() {
		It("benchmarks alpha (valid)", func() {
			s := "HelloWorld"
			experiment.SampleDuration("alpha/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = alphaRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("alpha/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks alpha (invalid)", func() {
			s := "Hello123"
			experiment.SampleDuration("alpha/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = alphaRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("alpha/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks alphanum (valid)", func() {
			s := "Hello123"
			experiment.SampleDuration("alphanum/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = alphanumRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("alphanum/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks numeric (valid)", func() {
			s := "1234567890"
			experiment.SampleDuration("numeric/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = numericRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("numeric/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})
	})

	Describe("@contains/@excludes", func() {
		It("benchmarks contains (valid)", func() {
			s := "hello world"
			experiment.SampleDuration("contains/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = strings.Contains(s, "world")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("contains/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 50*time.Microsecond))
		})

		It("benchmarks contains (invalid)", func() {
			s := "hello world"
			experiment.SampleDuration("contains/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = strings.Contains(s, "foo")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("contains/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})

		It("benchmarks excludes (valid)", func() {
			s := "hello world"
			experiment.SampleDuration("excludes/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = !strings.Contains(s, "foo")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("excludes/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})
	})

	Describe("@startswith/@endswith", func() {
		It("benchmarks startswith (valid)", func() {
			s := "https://example.com"
			experiment.SampleDuration("startswith/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = strings.HasPrefix(s, "https://")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("startswith/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})

		It("benchmarks startswith (invalid)", func() {
			s := "http://example.com"
			experiment.SampleDuration("startswith/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = strings.HasPrefix(s, "https://")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("startswith/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})

		It("benchmarks endswith (valid)", func() {
			s := "example.com"
			experiment.SampleDuration("endswith/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = strings.HasSuffix(s, ".com")
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("endswith/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 20*time.Microsecond))
		})
	})

	Describe("@regex", func() {
		It("benchmarks regex (valid)", func() {
			s := "AB-1234"
			experiment.SampleDuration("regex/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = customRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("regex/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks regex (invalid)", func() {
			s := "invalid"
			experiment.SampleDuration("regex/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = customRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("regex/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks complex regex", func() {
			complexRegex := regexp.MustCompile(
				`^(?:[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+)*|"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)\])$`,
			)
			s := "user@example.com"
			experiment.SampleDuration("regex/complex", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = complexRegex.MatchString(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("regex/complex")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 1*time.Millisecond))
		})
	})

	Describe("@format", func() {
		It("benchmarks format json (valid)", func() {
			s := `{"name":"test","value":123}`
			experiment.SampleDuration("format/json_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = json.Valid([]byte(s))
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("format/json_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 500*time.Microsecond))
		})

		It("benchmarks format json (invalid)", func() {
			s := `{"name":"test",invalid}`
			experiment.SampleDuration("format/json_invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_ = json.Valid([]byte(s))
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("format/json_invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 200*time.Microsecond))
		})

		It("benchmarks format yaml (valid)", func() {
			s := "name: test\nvalue: 123"
			experiment.SampleDuration("format/yaml_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					var v any
					_ = yaml.Unmarshal([]byte(s), &v)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("format/yaml_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 5*time.Millisecond))
		})

		It("benchmarks format yaml (invalid)", func() {
			s := "name: test\n  invalid indent"
			experiment.SampleDuration("format/yaml_invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					var v any
					_ = yaml.Unmarshal([]byte(s), &v)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("format/yaml_invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 5*time.Millisecond))
		})

		It("benchmarks format toml (valid)", func() {
			s := "name = \"test\"\nvalue = 123"
			experiment.SampleDuration("format/toml_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					var v any
					_ = toml.Unmarshal([]byte(s), &v)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("format/toml_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 5*time.Millisecond))
		})

		It("benchmarks format toml (invalid)", func() {
			s := "name = invalid toml"
			experiment.SampleDuration("format/toml_invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					var v any
					_ = toml.Unmarshal([]byte(s), &v)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("format/toml_invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 5*time.Millisecond))
		})

		It("benchmarks format csv (valid)", func() {
			s := "name,value\ntest,123"
			experiment.SampleDuration("format/csv_valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					r := csv.NewReader(strings.NewReader(s))
					_, _ = r.ReadAll()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("format/csv_valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 2*time.Millisecond))
		})

		It("benchmarks format csv (invalid)", func() {
			s := "name,value\n\"unclosed quote"
			experiment.SampleDuration("format/csv_invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					r := csv.NewReader(strings.NewReader(s))
					_, _ = r.ReadAll()
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("format/csv_invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 1*time.Millisecond))
		})
	})

	Context("Duration Validation", func() {
		It("benchmarks duration (valid)", func() {
			s := "1h30m"
			experiment.SampleDuration("duration/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_, _ = time.ParseDuration(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("duration/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks duration (invalid)", func() {
			s := "invalid"
			experiment.SampleDuration("duration/invalid", func(_ int) {
				for i := 0; i < 1000; i++ {
					_, _ = time.ParseDuration(s)
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("duration/invalid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks duration_min (valid)", func() {
			s := "5m"
			minNanos := int64(time.Minute) // 1m in nanoseconds
			experiment.SampleDuration("duration_min/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					if dur, err := time.ParseDuration(s); err == nil {
						_ = dur >= time.Duration(minNanos)
					}
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("duration_min/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks duration_max (valid)", func() {
			s := "30m"
			maxNanos := int64(time.Hour) // 1h in nanoseconds
			experiment.SampleDuration("duration_max/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					if dur, err := time.ParseDuration(s); err == nil {
						_ = dur <= time.Duration(maxNanos)
					}
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("duration_max/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})

		It("benchmarks duration combined (valid)", func() {
			s := "30m"
			minNanos := int64(time.Minute)  // 1m
			maxNanos := int64(time.Hour)    // 1h
			experiment.SampleDuration("duration_combined/valid", func(_ int) {
				for i := 0; i < 1000; i++ {
					if dur, err := time.ParseDuration(s); err == nil {
						_ = dur >= time.Duration(minNanos) && dur <= time.Duration(maxNanos)
					}
				}
			}, gmeasure.SamplingConfig{N: 100}, gmeasure.Precision(time.Nanosecond))

			stats := experiment.GetStats("duration_combined/valid")
			Expect(stats.DurationFor(gmeasure.StatMean)).To(BeNumerically("<", 100*time.Microsecond))
		})
	})
})

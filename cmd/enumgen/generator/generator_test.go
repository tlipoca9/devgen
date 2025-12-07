package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tlipoca9/devgen/cmd/enumgen/generator"
	"github.com/tlipoca9/devgen/genkit"
)

func TestGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Enumgen Generator Suite")
}

var _ = Describe("Generator", func() {
	var (
		gen *generator.Generator
	)

	BeforeEach(func() {
		gen = generator.New()
	})

	Describe("New", func() {
		It("should create a new generator", func() {
			g := generator.New()
			Expect(g).NotTo(BeNil())
		})
	})

	Describe("Name", func() {
		It("should return the correct tool name", func() {
			Expect(gen.Name()).To(Equal("enumgen"))
		})
	})

	Describe("TrimPrefix", func() {
		It("should trim type prefix from value name", func() {
			Expect(generator.TrimPrefix("StatusActive", "Status")).To(Equal("Active"))
			Expect(generator.TrimPrefix("StatusPending", "Status")).To(Equal("Pending"))
		})

		It("should return original name if prefix not found", func() {
			Expect(generator.TrimPrefix("Active", "Status")).To(Equal("Active"))
		})

		It("should return original name if result would be empty", func() {
			Expect(generator.TrimPrefix("Status", "Status")).To(Equal("Status"))
		})

		It("should handle empty strings", func() {
			Expect(generator.TrimPrefix("", "Status")).To(Equal(""))
			Expect(generator.TrimPrefix("StatusActive", "")).To(Equal("StatusActive"))
		})
	})

	Describe("GetValueName", func() {
		It("should return custom name from annotation", func() {
			ev := &genkit.EnumValue{
				Name: "StatusActive",
				Doc:  "enumgen:@name(active)",
			}
			Expect(generator.GetValueName(ev, "Status")).To(Equal("active"))
		})

		It("should use TrimPrefix when no annotation", func() {
			ev := &genkit.EnumValue{
				Name: "StatusActive",
				Doc:  "",
			}
			Expect(generator.GetValueName(ev, "Status")).To(Equal("Active"))
		})

		It("should handle comment annotation", func() {
			ev := &genkit.EnumValue{
				Name:    "StatusPending",
				Doc:     "enumgen:@name(pending)",
				Comment: "",
			}
			Expect(generator.GetValueName(ev, "Status")).To(Equal("pending"))
		})
	})

	Describe("FindEnums", func() {
		It("should find enums with annotation", func() {
			pkg := &genkit.Package{
				Enums: []*genkit.Enum{
					{Name: "Status", Doc: "enumgen:@enum(string)"},
					{Name: "Type", Doc: "some other doc"},
					{Name: "Kind", Doc: "enumgen:@enum(json, text)"},
				},
			}
			enums := gen.FindEnums(pkg)
			Expect(enums).To(HaveLen(2))
			Expect(enums[0].Name).To(Equal("Status"))
			Expect(enums[1].Name).To(Equal("Kind"))
		})

		It("should return empty slice when no enums have annotation", func() {
			pkg := &genkit.Package{
				Enums: []*genkit.Enum{
					{Name: "Status", Doc: "some doc"},
					{Name: "Type", Doc: "another doc"},
				},
			}
			enums := gen.FindEnums(pkg)
			Expect(enums).To(BeEmpty())
		})

		It("should return empty slice for empty package", func() {
			pkg := &genkit.Package{
				Enums: []*genkit.Enum{},
			}
			enums := gen.FindEnums(pkg)
			Expect(enums).To(BeEmpty())
		})
	})

	Describe("GenerateOption enum", func() {
		Describe("IsValid", func() {
			It("should return true for valid options", func() {
				Expect(generator.GenerateOptionString.IsValid()).To(BeTrue())
				Expect(generator.GenerateOptionJSON.IsValid()).To(BeTrue())
				Expect(generator.GenerateOptionText.IsValid()).To(BeTrue())
				Expect(generator.GenerateOptionSQL.IsValid()).To(BeTrue())
			})

			It("should return false for invalid options", func() {
				var invalid generator.GenerateOption = 100
				Expect(invalid.IsValid()).To(BeFalse())
			})
		})

		Describe("String", func() {
			It("should return correct string representation", func() {
				Expect(generator.GenerateOptionString.String()).To(Equal("string"))
				Expect(generator.GenerateOptionJSON.String()).To(Equal("json"))
				Expect(generator.GenerateOptionText.String()).To(Equal("text"))
				Expect(generator.GenerateOptionSQL.String()).To(Equal("sql"))
			})

			It("should return formatted string for invalid option", func() {
				var invalid generator.GenerateOption = 100
				Expect(invalid.String()).To(ContainSubstring("GenerateOption"))
				Expect(invalid.String()).To(ContainSubstring("100"))
			})
		})

		Describe("JSON marshaling", func() {
			It("should marshal to JSON string", func() {
				data, err := generator.GenerateOptionString.MarshalJSON()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal(`"string"`))
			})

			It("should unmarshal from JSON string", func() {
				var opt generator.GenerateOption
				err := opt.UnmarshalJSON([]byte(`"json"`))
				Expect(err).NotTo(HaveOccurred())
				Expect(opt).To(Equal(generator.GenerateOptionJSON))
			})

			It("should return error for invalid JSON", func() {
				var opt generator.GenerateOption
				err := opt.UnmarshalJSON([]byte(`invalid`))
				Expect(err).To(HaveOccurred())
			})

			It("should return error for unknown value", func() {
				var opt generator.GenerateOption
				err := opt.UnmarshalJSON([]byte(`"unknown"`))
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("Text marshaling", func() {
			It("should marshal to text", func() {
				data, err := generator.GenerateOptionText.MarshalText()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal("text"))
			})

			It("should unmarshal from text", func() {
				var opt generator.GenerateOption
				err := opt.UnmarshalText([]byte("sql"))
				Expect(err).NotTo(HaveOccurred())
				Expect(opt).To(Equal(generator.GenerateOptionSQL))
			})

			It("should return error for unknown text value", func() {
				var opt generator.GenerateOption
				err := opt.UnmarshalText([]byte("unknown"))
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("SQL Value/Scan", func() {
			It("should return driver value", func() {
				val, err := generator.GenerateOptionJSON.Value()
				Expect(err).NotTo(HaveOccurred())
				Expect(val).To(Equal("json"))
			})

			It("should scan from string", func() {
				var opt generator.GenerateOption
				err := opt.Scan("text")
				Expect(err).NotTo(HaveOccurred())
				Expect(opt).To(Equal(generator.GenerateOptionText))
			})

			It("should scan from []byte", func() {
				var opt generator.GenerateOption
				err := opt.Scan([]byte("sql"))
				Expect(err).NotTo(HaveOccurred())
				Expect(opt).To(Equal(generator.GenerateOptionSQL))
			})

			It("should handle nil scan", func() {
				var opt generator.GenerateOption
				err := opt.Scan(nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return error for unsupported type", func() {
				var opt generator.GenerateOption
				err := opt.Scan(123)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot scan"))
			})

			It("should return error for unknown value", func() {
				var opt generator.GenerateOption
				err := opt.Scan("unknown")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GenerateOptionEnums helper", func() {
		Describe("List", func() {
			It("should return all valid options", func() {
				list := generator.GenerateOptionEnums.List()
				Expect(list).To(HaveLen(4))
				Expect(list).To(ContainElements(
					generator.GenerateOptionString,
					generator.GenerateOptionJSON,
					generator.GenerateOptionText,
					generator.GenerateOptionSQL,
				))
			})
		})

		Describe("Contains", func() {
			It("should return true for valid options", func() {
				Expect(generator.GenerateOptionEnums.Contains(generator.GenerateOptionString)).To(BeTrue())
				Expect(generator.GenerateOptionEnums.Contains(generator.GenerateOptionJSON)).To(BeTrue())
			})

			It("should return false for invalid options", func() {
				var invalid generator.GenerateOption = 100
				Expect(generator.GenerateOptionEnums.Contains(invalid)).To(BeFalse())
			})
		})

		Describe("ContainsName", func() {
			It("should return true for valid names", func() {
				Expect(generator.GenerateOptionEnums.ContainsName("string")).To(BeTrue())
				Expect(generator.GenerateOptionEnums.ContainsName("json")).To(BeTrue())
				Expect(generator.GenerateOptionEnums.ContainsName("text")).To(BeTrue())
				Expect(generator.GenerateOptionEnums.ContainsName("sql")).To(BeTrue())
			})

			It("should return false for invalid names", func() {
				Expect(generator.GenerateOptionEnums.ContainsName("unknown")).To(BeFalse())
				Expect(generator.GenerateOptionEnums.ContainsName("")).To(BeFalse())
			})
		})

		Describe("Parse", func() {
			It("should parse valid names", func() {
				opt, err := generator.GenerateOptionEnums.Parse("string")
				Expect(err).NotTo(HaveOccurred())
				Expect(opt).To(Equal(generator.GenerateOptionString))

				opt, err = generator.GenerateOptionEnums.Parse("json")
				Expect(err).NotTo(HaveOccurred())
				Expect(opt).To(Equal(generator.GenerateOptionJSON))
			})

			It("should return error for invalid names", func() {
				_, err := generator.GenerateOptionEnums.Parse("unknown")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid GenerateOption"))
			})
		})

		Describe("Name", func() {
			It("should return name for valid options", func() {
				Expect(generator.GenerateOptionEnums.Name(generator.GenerateOptionString)).To(Equal("string"))
				Expect(generator.GenerateOptionEnums.Name(generator.GenerateOptionJSON)).To(Equal("json"))
			})

			It("should return formatted name for invalid options", func() {
				var invalid generator.GenerateOption = 100
				name := generator.GenerateOptionEnums.Name(invalid)
				Expect(name).To(ContainSubstring("GenerateOption"))
				Expect(name).To(ContainSubstring("100"))
			})
		})

		Describe("Names", func() {
			It("should return all valid names", func() {
				names := generator.GenerateOptionEnums.Names()
				Expect(names).To(HaveLen(4))
				Expect(names).To(ContainElements("string", "json", "text", "sql"))
			})
		})
	})

	Describe("Run method", func() {
		var (
			tempDir string
			gk      *genkit.Generator
		)

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "enumgen-run-test-*")
			Expect(err).NotTo(HaveOccurred())

			// Create go.mod
			goMod := filepath.Join(tempDir, "go.mod")
			err = os.WriteFile(goMod, []byte("module testpkg\n\ngo 1.21\n"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = os.RemoveAll(tempDir)
		})

		It("should run successfully with enums", func() {
			testFile := filepath.Join(tempDir, "status.go")
			content := `package testpkg

// Status represents a status.
// enumgen:@enum(string)
type Status int

const (
	StatusActive Status = iota + 1
	StatusInactive
)
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			log := genkit.NewLogger()
			err = gen.Run(gk, log)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should run successfully without enums", func() {
			testFile := filepath.Join(tempDir, "noenum.go")
			content := `package testpkg

type NoEnum int
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			log := genkit.NewLogger()
			err = gen.Run(gk, log)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error for duplicate enum names", func() {
			testFile := filepath.Join(tempDir, "dup.go")
			content := `package testpkg

// DupEnum has duplicate names.
// enumgen:@enum(string)
type DupEnum int

const (
	// enumgen:@name(same)
	DupEnumFirst DupEnum = iota + 1
	// enumgen:@name(same)
	DupEnumSecond
)
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			log := genkit.NewLogger()
			err = gen.Run(gk, log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate"))
		})
	})

	Describe("Integration tests", func() {
		var (
			tempDir string
			gk      *genkit.Generator
		)

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "enumgen-test-*")
			Expect(err).NotTo(HaveOccurred())

			// Create test source file
			testFile := filepath.Join(tempDir, "status.go")
			content := `package testpkg

// Status represents a status.
// enumgen:@enum(string, json)
type Status int

const (
	// enumgen:@name(active)
	StatusActive Status = iota + 1
	// enumgen:@name(inactive)
	StatusInactive
	// enumgen:@name(pending)
	StatusPending
)
`
			err = os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Create go.mod
			goMod := filepath.Join(tempDir, "go.mod")
			err = os.WriteFile(goMod, []byte("module testpkg\n\ngo 1.21\n"), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
		})

		AfterEach(func() {
			_ = os.RemoveAll(tempDir)
		})

		It("should process package and generate code", func() {
			err := gk.Load(".")
			Expect(err).NotTo(HaveOccurred())
			Expect(gk.Packages).To(HaveLen(1))

			err = gen.ProcessPackage(gk, gk.Packages[0])
			Expect(err).NotTo(HaveOccurred())

			files, err := gk.DryRun()
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1))

			// Check generated content
			for _, content := range files {
				code := string(content)
				Expect(code).To(ContainSubstring("Code generated by enumgen"))
				Expect(code).To(ContainSubstring("package testpkg"))
				Expect(code).To(ContainSubstring("func (x Status) IsValid() bool"))
				Expect(code).To(ContainSubstring("func (x Status) String() string"))
				Expect(code).To(ContainSubstring("func (x Status) MarshalJSON()"))
				Expect(code).To(ContainSubstring("func (x *Status) UnmarshalJSON(data []byte)"))
				Expect(code).To(ContainSubstring("StatusEnums"))
				Expect(code).To(ContainSubstring(`"active"`))
				Expect(code).To(ContainSubstring(`"inactive"`))
				Expect(code).To(ContainSubstring(`"pending"`))
			}
		})

		It("should handle duplicate enum names", func() {
			// Create test file with duplicate names
			testFile := filepath.Join(tempDir, "duplicate.go")
			content := `package testpkg

// DupEnum has duplicate names.
// enumgen:@enum(string)
type DupEnum int

const (
	// enumgen:@name(same)
	DupEnumFirst DupEnum = iota + 1
	// enumgen:@name(same)
	DupEnumSecond
)
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk2 := genkit.New(genkit.Options{Dir: tempDir})
			err = gk2.Load(".")
			Expect(err).NotTo(HaveOccurred())

			// Find the DupEnum
			var dupEnum *genkit.Enum
			for _, e := range gk2.Packages[0].Enums {
				if e.Name == "DupEnum" {
					dupEnum = e
					break
				}
			}
			Expect(dupEnum).NotTo(BeNil())

			// Generate should fail with duplicate name error
			outPath := genkit.OutputPath(tempDir, "testpkg_enum.go")
			g := gk2.NewGeneratedFile(outPath, gk2.Packages[0].GoImportPath())
			gen.WriteHeader(g, "testpkg")
			err = gen.GenerateEnum(g, dupEnum)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate"))
		})

		It("should generate all options (string, json, text, sql)", func() {
			testFile := filepath.Join(tempDir, "allopts.go")
			content := `package testpkg

// AllOpts has all options.
// enumgen:@enum(string, json, text, sql)
type AllOpts int

const (
	AllOptsOne AllOpts = iota + 1
	AllOptsTwo
)
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk2 := genkit.New(genkit.Options{Dir: tempDir})
			err = gk2.Load(".")
			Expect(err).NotTo(HaveOccurred())

			err = gen.ProcessPackage(gk2, gk2.Packages[0])
			Expect(err).NotTo(HaveOccurred())

			files, err := gk2.DryRun()
			Expect(err).NotTo(HaveOccurred())

			for _, content := range files {
				code := string(content)
				// Check all methods are generated
				Expect(code).To(ContainSubstring("func (x AllOpts) String()"))
				Expect(code).To(ContainSubstring("func (x AllOpts) MarshalJSON()"))
				Expect(code).To(ContainSubstring("func (x *AllOpts) UnmarshalJSON"))
				Expect(code).To(ContainSubstring("func (x AllOpts) MarshalText()"))
				Expect(code).To(ContainSubstring("func (x *AllOpts) UnmarshalText"))
				Expect(code).To(ContainSubstring("func (x AllOpts) Value()"))
				Expect(code).To(ContainSubstring("func (x *AllOpts) Scan"))
			}
		})

		It("should skip package without enums", func() {
			// Create a file without enum annotations
			testFile := filepath.Join(tempDir, "noenum.go")
			content := `package testpkg

type NoEnum int

const (
	NoEnumOne NoEnum = iota + 1
	NoEnumTwo
)
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk2 := genkit.New(genkit.Options{Dir: tempDir})
			err = gk2.Load(".")
			Expect(err).NotTo(HaveOccurred())

			// Find package without annotated enums
			var noEnumPkg *genkit.Package
			for _, pkg := range gk2.Packages {
				hasAnnotated := false
				for _, e := range pkg.Enums {
					if genkit.HasAnnotation(e.Doc, "enumgen", "enum") {
						hasAnnotated = true
						break
					}
				}
				if !hasAnnotated {
					noEnumPkg = pkg
					break
				}
			}

			if noEnumPkg != nil {
				err = gen.ProcessPackage(gk2, noEnumPkg)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})

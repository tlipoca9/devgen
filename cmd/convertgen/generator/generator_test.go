package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tlipoca9/devgen/cmd/convertgen/generator"
	"github.com/tlipoca9/devgen/genkit"
)

func TestGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Convertgen Generator Suite")
}

var _ = Describe("Generator", func() {
	var gen *generator.Generator

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
			Expect(gen.Name()).To(Equal("convertgen"))
		})
	})

	Describe("Config", func() {
		It("should return correct output suffix", func() {
			config := gen.Config()
			Expect(config.OutputSuffix).To(Equal("_convertgen.go"))
		})

		It("should have all required annotations", func() {
			config := gen.Config()
			names := make([]string, len(config.Annotations))
			for i, ann := range config.Annotations {
				names[i] = ann.Name
			}
			Expect(names).To(ContainElements("converter", "map", "ignore", "shallow"))
		})
	})

	Describe("Rules", func() {
		It("should return rules with correct name", func() {
			rules := gen.Rules()
			Expect(rules).To(HaveLen(1))
			Expect(rules[0].Name).To(Equal("devgen-tool-convertgen"))
		})
	})

	Describe("Nested Conversion", func() {
		var tempDir string
		var gk *genkit.Generator

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "convertgen-test-*")
			Expect(err).NotTo(HaveOccurred())

			goMod := filepath.Join(tempDir, "go.mod")
			err = os.WriteFile(goMod, []byte("module testpkg\n\ngo 1.21\n"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = os.RemoveAll(tempDir)
		})

		It("should reuse ConvertAddress method for nested struct", func() {
			testFile := filepath.Join(tempDir, "types.go")
			content := `package testpkg

type Address struct {
    Street string
    City   string
}

type AddressDTO struct {
    Street string
    City   string
}

type User struct {
    Name    string
    Address *Address
}

type UserDTO struct {
    Name    string
    Address *AddressDTO
}

// convertgen:@converter
type UserConverter interface {
    ConvertAddress(*Address) *AddressDTO
    Convert(*User) *UserDTO
}
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			log := genkit.NewLogger()
			err = gen.Run(gk, log)
			Expect(err).NotTo(HaveOccurred())

			files, err := gk.DryRun()
			Expect(err).NotTo(HaveOccurred())

			for _, content := range files {
				code := string(content)
				// Verify generated code uses ConvertAddress method
				Expect(code).To(ContainSubstring("c.ConvertAddress(src.Address)"))
			}
		})

		It("should reuse Convert method for slice elements", func() {
			testFile := filepath.Join(tempDir, "types.go")
			content := `package testpkg

type Item struct {
    ID   int
    Name string
}

type ItemDTO struct {
    ID   int
    Name string
}

// convertgen:@converter
type ItemConverter interface {
    Convert(*Item) *ItemDTO
    ConvertList([]*Item) []*ItemDTO
}
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			log := genkit.NewLogger()
			err = gen.Run(gk, log)
			Expect(err).NotTo(HaveOccurred())

			files, err := gk.DryRun()
			Expect(err).NotTo(HaveOccurred())

			for _, content := range files {
				code := string(content)
				// Verify ConvertList uses Convert method
				Expect(code).To(ContainSubstring("c.Convert(v)"))
			}
		})

		It("should handle field mapping correctly", func() {
			testFile := filepath.Join(tempDir, "types.go")
			content := `package testpkg

type User struct {
    ID       int
    FullName string
}

type UserDTO struct {
    ID   int
    Name string
}

// convertgen:@converter
type UserConverter interface {
    // convertgen:@map(FullName, Name)
    Convert(*User) *UserDTO
}
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			log := genkit.NewLogger()
			err = gen.Run(gk, log)
			Expect(err).NotTo(HaveOccurred())

			files, err := gk.DryRun()
			Expect(err).NotTo(HaveOccurred())

			for _, content := range files {
				code := string(content)
				// Verify field mapping
				Expect(code).To(ContainSubstring("dst.Name = src.FullName"))
			}
		})

		It("should handle ignore annotation correctly", func() {
			testFile := filepath.Join(tempDir, "types.go")
			content := `package testpkg

type User struct {
    ID       int
    Name     string
    Password string
}

type UserDTO struct {
    ID   int
    Name string
}

// convertgen:@converter
type UserConverter interface {
    // convertgen:@ignore(Password)
    Convert(*User) *UserDTO
}
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			log := genkit.NewLogger()
			err = gen.Run(gk, log)
			Expect(err).NotTo(HaveOccurred())

			files, err := gk.DryRun()
			Expect(err).NotTo(HaveOccurred())

			for _, content := range files {
				code := string(content)
				// Verify Password is not in generated code
				Expect(code).NotTo(ContainSubstring("Password"))
			}
		})
	})

	Describe("Validate", func() {
		var tempDir string
		var gk *genkit.Generator

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "convertgen-validate-test-*")
			Expect(err).NotTo(HaveOccurred())

			goMod := filepath.Join(tempDir, "go.mod")
			err = os.WriteFile(goMod, []byte("module testpkg\n\ngo 1.21\n"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = os.RemoveAll(tempDir)
		})

		It("should report error for invalid @map source field", func() {
			testFile := filepath.Join(tempDir, "types.go")
			content := `package testpkg

type User struct {
    Name string
}

type UserDTO struct {
    DisplayName string
}

// convertgen:@converter
type UserConverter interface {
    // convertgen:@map(NonExistent, DisplayName)
    Convert(*User) *UserDTO
}
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			diagnostics := gen.Validate(gk, genkit.NewLogger())
			Expect(diagnostics).NotTo(BeEmpty())

			hasFieldNotFound := false
			for _, d := range diagnostics {
				if d.Code == "E003" && strings.Contains(d.Message, "NonExistent") {
					hasFieldNotFound = true
					break
				}
			}
			Expect(hasFieldNotFound).To(BeTrue())
		})

		It("should report error for invalid @map destination field", func() {
			testFile := filepath.Join(tempDir, "types.go")
			content := `package testpkg

type User struct {
    Name string
}

type UserDTO struct {
    DisplayName string
}

// convertgen:@converter
type UserConverter interface {
    // convertgen:@map(Name, NonExistent)
    Convert(*User) *UserDTO
}
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			diagnostics := gen.Validate(gk, genkit.NewLogger())
			Expect(diagnostics).NotTo(BeEmpty())

			hasFieldNotFound := false
			for _, d := range diagnostics {
				if d.Code == "E003" && strings.Contains(d.Message, "NonExistent") {
					hasFieldNotFound = true
					break
				}
			}
			Expect(hasFieldNotFound).To(BeTrue())
		})

		It("should warn for missing destination field mapping", func() {
			testFile := filepath.Join(tempDir, "types.go")
			content := `package testpkg

type User struct {
    Name string
}

type UserDTO struct {
    Name        string
    DisplayName string
}

// convertgen:@converter
type UserConverter interface {
    Convert(*User) *UserDTO
}
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			diagnostics := gen.Validate(gk, genkit.NewLogger())

			hasWarning := false
			for _, d := range diagnostics {
				if d.Code == "W001" && d.Severity == genkit.DiagnosticWarning {
					hasWarning = true
					break
				}
			}
			Expect(hasWarning).To(BeTrue())
		})

		It("should pass validation for correct converter", func() {
			testFile := filepath.Join(tempDir, "types.go")
			content := `package testpkg

type User struct {
    ID   int
    Name string
}

type UserDTO struct {
    ID   int
    Name string
}

// convertgen:@converter
type UserConverter interface {
    Convert(*User) *UserDTO
}
`
			err := os.WriteFile(testFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			gk = genkit.New(genkit.Options{Dir: tempDir})
			err = gk.Load(".")
			Expect(err).NotTo(HaveOccurred())

			diagnostics := gen.Validate(gk, genkit.NewLogger())

			// Should not have any error-level diagnostics
			for _, d := range diagnostics {
				Expect(d.Severity).NotTo(Equal(genkit.DiagnosticError))
			}
		})
	})
})

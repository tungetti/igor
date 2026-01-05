package distro

import (
	"context"
	"testing"
	"time"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/exec"
)

// ============================================================================
// Benchmark Tests - Performance Testing for Distribution Detection
// ============================================================================

// BenchmarkDetector_Detect benchmarks the main detection function.
func BenchmarkDetector_Detect(b *testing.B) {
	mockFS := NewMockFileReader(map[string][]byte{
		"/etc/os-release": []byte(ubuntu2404OSRelease),
	})
	mockExec := exec.NewMockExecutor()
	detector := NewDetector(mockExec, mockFS)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.Detect(ctx)
	}
}

// BenchmarkDetector_Detect_AllDistros benchmarks detection for different distros.
func BenchmarkDetector_Detect_AllDistros(b *testing.B) {
	distros := map[string]string{
		"Ubuntu":    ubuntu2404OSRelease,
		"Fedora":    fedora40OSRelease,
		"Arch":      archOSRelease,
		"Debian":    debian12OSRelease,
		"Rocky":     rocky9OSRelease,
		"openSUSE":  opensuse155OSRelease,
		"Manjaro":   manjaroOSRelease,
		"Pop_OS":    popOSRelease,
		"LinuxMint": linuxMintOSRelease,
	}

	for name, content := range distros {
		b.Run(name, func(b *testing.B) {
			mockFS := NewMockFileReader(map[string][]byte{
				"/etc/os-release": []byte(content),
			})
			mockExec := exec.NewMockExecutor()
			detector := NewDetector(mockExec, mockFS)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = detector.Detect(ctx)
			}
		})
	}
}

// BenchmarkParseOSReleaseContent benchmarks os-release parsing.
func BenchmarkParseOSReleaseContent(b *testing.B) {
	content := ubuntu2404OSRelease

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseOSReleaseContent(content)
	}
}

// BenchmarkParseOSReleaseContent_LargeFile benchmarks parsing a large os-release file.
func BenchmarkParseOSReleaseContent_LargeFile(b *testing.B) {
	// Create a larger os-release file with many fields
	content := `PRETTY_NAME="Ubuntu 24.04 LTS"
NAME="Ubuntu"
VERSION_ID="24.04"
VERSION="24.04 LTS (Noble Numbat)"
VERSION_CODENAME=noble
ID=ubuntu
ID_LIKE=debian
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
UBUNTU_CODENAME=noble
LOGO=ubuntu-logo
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:canonical:ubuntu_linux:24.04"
VARIANT="Desktop"
VARIANT_ID=desktop
DOCUMENTATION_URL="https://help.ubuntu.com/"
EXTRA_FIELD_1="extra value 1"
EXTRA_FIELD_2="extra value 2"
EXTRA_FIELD_3="extra value 3"
EXTRA_FIELD_4="extra value 4"
EXTRA_FIELD_5="extra value 5"
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseOSReleaseContent(content)
	}
}

// BenchmarkParseLSBReleaseContent benchmarks lsb-release parsing.
func BenchmarkParseLSBReleaseContent(b *testing.B) {
	content := `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=22.04
DISTRIB_CODENAME=jammy
DISTRIB_DESCRIPTION="Ubuntu 22.04.3 LTS"
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseLSBReleaseContent(content)
	}
}

// BenchmarkDetectFamily benchmarks the family detection function.
func BenchmarkDetectFamily(b *testing.B) {
	testCases := []struct {
		name   string
		id     string
		idLike []string
	}{
		{"Ubuntu", "ubuntu", []string{"debian"}},
		{"Fedora", "fedora", nil},
		{"Arch", "arch", nil},
		{"Rocky", "rocky", []string{"rhel", "centos", "fedora"}},
		{"Unknown", "customdistro", []string{"otherdistro"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = DetectFamily(tc.id, tc.idLike)
			}
		})
	}
}

// BenchmarkDetector_Parallel benchmarks parallel detection.
func BenchmarkDetector_Parallel(b *testing.B) {
	mockFS := NewMockFileReader(map[string][]byte{
		"/etc/os-release": []byte(ubuntu2404OSRelease),
	})
	mockExec := exec.NewMockExecutor()
	detector := NewDetector(mockExec, mockFS)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = detector.Detect(ctx)
		}
	})
}

// BenchmarkDistribution_String benchmarks the String() method.
func BenchmarkDistribution_String(b *testing.B) {
	testCases := []struct {
		name string
		dist Distribution
	}{
		{
			"WithPrettyName",
			Distribution{PrettyName: "Ubuntu 24.04 LTS"},
		},
		{
			"WithNameAndVersion",
			Distribution{Name: "Ubuntu", VersionID: "24.04"},
		},
		{
			"WithNameOnly",
			Distribution{Name: "Ubuntu"},
		},
		{
			"WithIDOnly",
			Distribution{ID: "ubuntu"},
		},
		{
			"Empty",
			Distribution{},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = tc.dist.String()
			}
		})
	}
}

// BenchmarkDistribution_IsFamily benchmarks family checking methods.
func BenchmarkDistribution_IsFamily(b *testing.B) {
	families := []struct {
		name   string
		family constants.DistroFamily
	}{
		{"Debian", constants.FamilyDebian},
		{"RHEL", constants.FamilyRHEL},
		{"Arch", constants.FamilyArch},
		{"SUSE", constants.FamilySUSE},
		{"Unknown", constants.FamilyUnknown},
	}

	for _, f := range families {
		dist := Distribution{Family: f.family}
		b.Run(f.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = dist.IsDebian()
				_ = dist.IsRHEL()
				_ = dist.IsArch()
				_ = dist.IsSUSE()
				_ = dist.IsUnknown()
			}
		})
	}
}

// BenchmarkDistribution_MajorVersion benchmarks the MajorVersion method.
func BenchmarkDistribution_MajorVersion(b *testing.B) {
	testCases := []struct {
		name      string
		versionID string
	}{
		{"Simple", "24.04"},
		{"Complex", "9.3.1"},
		{"Dash", "15-sp5"},
		{"Single", "40"},
		{"Empty", ""},
	}

	for _, tc := range testCases {
		dist := Distribution{VersionID: tc.versionID}
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = dist.MajorVersion()
			}
		})
	}
}

// BenchmarkDistribution_MinorVersion benchmarks the MinorVersion method.
func BenchmarkDistribution_MinorVersion(b *testing.B) {
	testCases := []struct {
		name      string
		versionID string
	}{
		{"Simple", "24.04"},
		{"Complex", "9.3.1"},
		{"Dash", "15-sp5"},
		{"Single", "40"},
		{"Empty", ""},
	}

	for _, tc := range testCases {
		dist := Distribution{VersionID: tc.versionID}
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = dist.MinorVersion()
			}
		})
	}
}

// BenchmarkDistribution_IsRolling benchmarks the IsRolling method.
func BenchmarkDistribution_IsRolling(b *testing.B) {
	testCases := []struct {
		name string
		dist Distribution
	}{
		{
			"Arch",
			Distribution{ID: "arch", Family: constants.FamilyArch},
		},
		{
			"Tumbleweed",
			Distribution{ID: "opensuse-tumbleweed", Family: constants.FamilySUSE},
		},
		{
			"Ubuntu",
			Distribution{ID: "ubuntu", Family: constants.FamilyDebian, VersionID: "24.04"},
		},
		{
			"BuildIDOnly",
			Distribution{ID: "customrolling", BuildID: "20240101"},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = tc.dist.IsRolling()
			}
		})
	}
}

// BenchmarkDetector_WithFallback benchmarks detection with fallback files.
func BenchmarkDetector_WithFallback(b *testing.B) {
	b.Run("OSRelease", func(b *testing.B) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(ubuntu2404OSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = detector.Detect(ctx)
		}
	})

	b.Run("SecondaryOSRelease", func(b *testing.B) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/usr/lib/os-release": []byte(fedora40OSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = detector.Detect(ctx)
		}
	})

	b.Run("LSBRelease", func(b *testing.B) {
		lsbRelease := `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=22.04
DISTRIB_CODENAME=jammy
DISTRIB_DESCRIPTION="Ubuntu 22.04 LTS"
`
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/lsb-release": []byte(lsbRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = detector.Detect(ctx)
		}
	})

	b.Run("RedhatRelease", func(b *testing.B) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/redhat-release": []byte("Rocky Linux release 9.3 (Blue Onyx)"),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = detector.Detect(ctx)
		}
	})

	b.Run("DebianVersion", func(b *testing.B) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/debian_version": []byte("12.4"),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = detector.Detect(ctx)
		}
	})
}

// BenchmarkDetector_ContextWithTimeout benchmarks detection with context timeout.
func BenchmarkDetector_ContextWithTimeout(b *testing.B) {
	mockFS := NewMockFileReader(map[string][]byte{
		"/etc/os-release": []byte(ubuntu2404OSRelease),
	})
	mockExec := exec.NewMockExecutor()
	detector := NewDetector(mockExec, mockFS)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = detector.Detect(ctx)
		cancel()
	}
}

// BenchmarkUnquote benchmarks the unquote function.
func BenchmarkUnquote(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"DoubleQuoted", `"Ubuntu 24.04 LTS"`},
		{"SingleQuoted", `'Ubuntu 24.04 LTS'`},
		{"Unquoted", `Ubuntu`},
		{"Short", `a`},
		{"Empty", ``},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = unquote(tc.input)
			}
		})
	}
}

// BenchmarkParseIDLike benchmarks the parseIDLike function.
func BenchmarkParseIDLike(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"Single", "debian"},
		{"Multiple", "ubuntu debian"},
		{"Many", "rhel centos fedora"},
		{"Empty", ""},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = parseIDLike(tc.input)
			}
		})
	}
}
